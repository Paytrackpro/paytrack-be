package service

import (
	"fmt"

	"github.com/Paytrackpro/paytrack-be/authpb"
	"github.com/Paytrackpro/paytrack-be/storage"
	"github.com/Paytrackpro/paytrack-be/utils"
	"github.com/Paytrackpro/paytrack-be/webserver/portal"
	socketio "github.com/googollee/go-socket.io"
	"gorm.io/gorm"
)

type Config struct {
	Exchange        string `yaml:"exchange"`
	ExchangeList    string `yaml:"allowexchanges"`
	CoimarketcapKey string `yaml:"coimarketcapKey"`
	AuthType        int    `yaml:"authType"`
	AuthHost        string `yaml:"authHost"`
	BaseUrl         string `yaml:"baseUrl"`
}

type Service struct {
	db              *gorm.DB
	Conf            Config
	exchange        string
	ExchangeList    string
	coinMaketCapKey string
	timeState       *actionTimeState
	socket          *socketio.Server
	AuthClient      *authpb.AuthServiceClient
}

func NewService(conf Config, db *gorm.DB, socket *socketio.Server) *Service {
	var authClient *authpb.AuthServiceClient
	if conf.AuthType == int(storage.AuthMicroservicePasskey) {
		authClient = InitAuthClient(conf.AuthHost)
	}
	return &Service{
		db:              db,
		Conf:            conf,
		exchange:        conf.Exchange,
		coinMaketCapKey: conf.CoimarketcapKey,
		ExchangeList:    conf.ExchangeList,
		timeState:       NewActionTime(),
		socket:          socket,
		AuthClient:      authClient,
	}
}

func (s *Service) ApprovePaymentRequest(id, userId uint64) (*storage.Payment, error) {
	var payment storage.Payment
	if err := s.db.First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, err
	}
	for i, approver := range payment.Approvers {
		if approver.ApproverId == userId {
			payment.Approvers[i].IsApproved = true
		}
	}

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

func (s *Service) GetSettingOfApprover(id uint64) ([]storage.ApproverSettings, error) {
	approvers := make([]storage.ApproverSettings, 0)
	if err := s.db.Where("approver_id = ?", id).Find(&approvers).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return approvers, nil
		}
		return nil, err
	}
	return approvers, nil
}

func (s *Service) GetApprovalSetting(sendId, recipientId, approverId uint64) (*storage.ApproverSettings, error) {
	var apst storage.ApproverSettings
	if err := s.db.First(&apst, "approver_id = ? AND recipient_id = ? AND send_user_id = ?", approverId, recipientId, sendId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &apst, nil
}

func (s *Service) GetApproverForPayment(sendId, recipientId uint64) ([]storage.ApproverSettings, error) {
	apst := make([]storage.ApproverSettings, 0)
	if err := s.db.Where("recipient_id = ? AND send_user_id = ?", recipientId, sendId).Find(&apst).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return apst, nil
}

func (s *Service) UpdateApproverSetting(userId uint64, approvers []portal.ApproversSettingRequest) ([]storage.ApproverSettings, error) {
	//Get all user is sender and approvers
	userIds := make([]uint64, 0)
	for _, approver := range approvers {
		userIds = append(userIds, approver.SendUserId)
		userIds = append(userIds, approver.ApproverIds...)
	}

	var users []storage.User
	//get all user on setting
	if err := s.db.Where("id IN ?", userIds).Find(&users).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(fmt.Errorf("sender user or approver user not found"), utils.ErrorBadRequest)
		}
		return nil, err
	}

	userMap := make(map[uint64]storage.User, 0)
	//conver slices user to map
	for _, user := range users {
		userMap[user.Id] = user
	}

	approversMap := make(map[uint64]storage.ApproverSettings, 0)
	settingApprovers := make([]storage.ApproverSettings, 0)
	for _, setting := range approvers {
		for _, v := range setting.ApproverIds {
			app := storage.ApproverSettings{
				ApproverId:   v,
				SendUserId:   setting.SendUserId,
				RecipientId:  userId,
				ApproverName: userMap[v].UserName,
				SendUserName: userMap[setting.SendUserId].UserName,
				ShowCost:     setting.ShowCost,
			}
			approversMap[v] = app
			settingApprovers = append(settingApprovers, app)
		}
	}

	//conver slices user to map
	for _, user := range users {
		userMap[user.Id] = user
	}

	// update all payment pendding
	payments := make([]*storage.Payment, 0)
	if err := s.db.Where("receiver_id = ? AND status = ?", userId, storage.PaymentStatusSent).Find(&payments).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	//update approver for all payment
	for i, payment := range payments {
		approverMap := make(map[uint64]storage.Approver, 0)
		for _, approver := range payment.Approvers {
			approverMap[approver.ApproverId] = approver
		}
		newApprovers := make([]storage.Approver, 0)
		for _, app := range settingApprovers {
			//if payment sender is not approval setting sender, move to the next iteration
			if payment.SenderId != app.SendUserId {
				continue
			}
			approved := false
			ap, ok := approverMap[app.ApproverId]
			if ok {
				approved = ap.IsApproved
			}
			newApprovers = append(newApprovers, storage.Approver{
				ApproverId:   app.ApproverId,
				ApproverName: app.ApproverName,
				IsApproved:   approved,
				ShowCost:     app.ShowCost,
			})
		}
		payments[i].Approvers = newApprovers
	}

	// Save to DB
	tx := s.db.Begin()

	// delete all old approver setting
	if err := tx.Where("recipient_id", userId).Delete(storage.ApproverSettings{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(settingApprovers) > 0 {
		// save new data
		if err := tx.Create(&settingApprovers).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// update payment
	if len(payments) > 0 {
		if err := tx.Save(&payments).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	tx.Commit()
	return settingApprovers, nil
}
