package webserver

import (
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
)

type apiOrder struct {
	*WebServer
}

func (a *apiOrder) getOrderManagement(w http.ResponseWriter, r *http.Request) {
	userInfo, _ := a.credentialsInfo(r)
	orderDisplayData, err := a.service.GetOrderManagement(userInfo.Id)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, orderDisplayData)
}

func (a *apiOrder) getMyOrders(w http.ResponseWriter, r *http.Request) {
	userInfo, _ := a.credentialsInfo(r)
	orderDisplayData, err := a.service.GetMyOrders(userInfo.Id)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, orderDisplayData)
}

func (a *apiOrder) getOrderDetail(w http.ResponseWriter, r *http.Request) {
	var id = chi.URLParam(r, "id")
	orderData, err := a.service.GetOrderDetail(utils.Uint64(id))
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}
	utils.ResponseOK(w, orderData)
}

// Get cart list
func (a *apiOrder) getOrderList(w http.ResponseWriter, r *http.Request) {
	userInfo, _ := a.credentialsInfo(r)

	carts, err := a.service.GetCartsForUser(userInfo.Id)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusBadGateway, err, nil)
		return
	}
	//handler for get cart information and return
	var data = map[uint64][]portal.CartDisplayData{}
	var ownerIdArr []uint64
	for _, cart := range carts {
		if !utils.ContainsUint64(ownerIdArr, cart.OwnerId) {
			ownerIdArr = append(ownerIdArr, cart.OwnerId)
		}
		var cartData []portal.CartDisplayData
		if data[cart.OwnerId] != nil {
			cartData = data[cart.OwnerId]
		}
		var product storage.Product
		if err := a.db.GetDB().Where("id = ?", cart.ProductId).First(&product).Error; err != nil {
			utils.Response(w, http.StatusBadGateway, err, nil)
			return
		}
		var tmpCartData portal.CartDisplayData
		tmpCartData.OwnerId = product.OwnerId
		tmpCartData.OwnerName = product.OwnerName
		tmpCartData.Quantity = cart.Quantity
		tmpCartData.Price = product.Price
		tmpCartData.Currency = product.Currency
		tmpCartData.ProductName = product.ProductName
		tmpCartData.ProductId = product.Id
		tmpCartData.AvatarBase64 = utils.ConvertImageToBase64(product.Avatar)
		tmpCartData.Stock = product.Stock
		cartData = append(cartData, tmpCartData)
		data[cart.OwnerId] = cartData
	}
	utils.ResponseOK(w, Map{
		"ownerIdArr": ownerIdArr,
		"cartData":   data,
	})
}

func (a *apiOrder) countOrder(w http.ResponseWriter, r *http.Request) {
	userInfo, _ := a.credentialsInfo(r)

	count, err := a.service.CountCartForUser(userInfo.Id)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusBadGateway, err, nil)
		return
	}
	utils.ResponseOK(w, count)

}

func (a *apiOrder) createOrders(w http.ResponseWriter, r *http.Request) {
	var body portal.OrderForm
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	userInfo, _ := a.credentialsInfo(r)
	orders, err := a.service.CreateOrders(userInfo.Id, userInfo.UserName, userInfo.DisplayName, body)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusOK, err, nil)
		return
	}
	res := Map{
		"orders": orders,
	}
	utils.ResponseOK(w, res, nil)
}

func (a *apiOrder) deleteOrder(w http.ResponseWriter, r *http.Request) {
	productId := r.FormValue("productId")
	userInfo, _ := a.credentialsInfo(r)
	if utils.IsEmpty(productId) {
		return
	}
	a.db.GetDB().Where("user_id = ? AND product_id = ?", userInfo.Id, productId).Delete(&storage.Cart{})
	utils.ResponseOK(w, nil)
}
