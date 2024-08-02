package webserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/btcpay"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
)

type apiBTCPay struct {
	*WebServer
}

func (a *apiBTCPay) markPaymentPaid(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentBTCPayInvoice
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	//get payment from ID
	var payment storage.Payment
	var filter = storage.PaymentFilter{
		Ids: []uint64{f.Id},
	}
	if err := a.db.First(&filter, &payment); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	claims, ok := a.credentialsInfo(r)
	if !ok || claims.Id != payment.ReceiverId {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("check handler permission failed"), utils.ErrorBadRequest), nil)
		return
	}

	if utils.IsEmpty(payment.BtcPayInvoiceId) || utils.IsEmpty(payment.BtcPayStoreId) {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("check invoice id or store id in payment failed"), utils.ErrorBadRequest), nil)
		return
	}

	//if payment is paid, return
	if payment.Status == storage.PaymentStatusPaid {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("the payment was marked as paid"), utils.ErrorBadRequest), nil)
		return
	}
	//Get store ID and apikey from shop user ID
	_, btcKey, storeErr := a.service.GetStoreIdAndApikeyFromUser(payment.SenderId)
	if storeErr != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(storeErr, utils.ErrorBadRequest), nil)
		return
	}
	btcpayClient := btcpay.NewBasicClient(a.conf.BtcPay.URL, btcKey)
	invoiceRes, statusCode, err := btcpayClient.GetInvoice(context.Background(), (*btcpay.StoreID)(&payment.BtcPayStoreId), (*btcpay.InvoiceID)(&payment.BtcPayInvoiceId))
	if err != nil || statusCode != http.StatusOK {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("get invoice on BTCPay failed"), utils.ErrorBadRequest), nil)
		return
	}
	if invoiceRes.Status != btcpay.GetInvoiceStatus().Settled && invoiceRes.Status != btcpay.GetInvoiceStatus().Processing {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("payment for invoice on btcpay not completed"), utils.ErrorBadRequest), nil)
		return
	}
	payment.PaidAt = time.Now()
	payment.Status = storage.PaymentStatusPaid
	payment.PaidBy = int(storage.PaidByBTCPay)
	payment.CheckoutLink = invoiceRes.CheckoutLink
	if err = a.db.Save(&payment); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.InternalError.With(err), nil)
		return
	}
	a.reloadList([]string{fmt.Sprint(payment.ReceiverId)}, "")
	utils.ResponseOK(w, payment)
}

func (a *apiBTCPay) reloadList(rooms []string, data interface{}) {
	for _, room := range rooms {
		a.socket.BroadcastToRoom("", room, "reloadList", data)
	}
}

func (a *apiBTCPay) createInvoice(w http.ResponseWriter, r *http.Request) {
	var f portal.PaymentBTCPayInvoice
	err := a.parseJSON(r, &f)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	//get payment from ID
	var payment storage.Payment
	var filter = storage.PaymentFilter{
		Ids: []uint64{f.Id},
	}
	if err := a.db.First(&filter, &payment); err != nil {
		utils.Response(w, http.StatusNotFound, utils.NotFoundError, nil)
		return
	}
	claims, ok := a.credentialsInfo(r)
	if !ok || claims.Id != payment.ReceiverId {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("check handler permission failed"), utils.ErrorBadRequest), nil)
		return
	}
	//if payment is paid, return
	if payment.Status == storage.PaymentStatusPaid {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("the payment was marked as paid"), utils.ErrorBadRequest), nil)
		return
	}
	//create invoice request
	invoiceReq := &btcpay.InvoiceRequest{
		Amount:   fmt.Sprintf("%f", payment.Amount),
		Currency: "USD",
		InvoiceCheckout: btcpay.InvoiceCheckout{
			RedirectURL: "/",
		},
	}
	//Get store ID and apikey from shop user ID
	storeId, btcKey, storeErr := a.service.GetStoreIdAndApikeyFromUser(payment.SenderId)
	if storeErr != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(storeErr, utils.ErrorBadRequest), nil)
		return
	}
	btcpayClient := btcpay.NewBasicClient(a.conf.BtcPay.URL, btcKey)

	invoiceRes, statusCode, err := btcpayClient.CreateInvoice(context.Background(), (*btcpay.StoreID)(&storeId), invoiceReq)

	if err != nil || statusCode != http.StatusOK {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("create invoice on BTCPay failed"), utils.ErrorBadRequest), nil)
		return
	}
	//update storeID and invoicesID in payment
	payment.BtcPayInvoiceId = string(invoiceRes.ID)
	payment.BtcPayStoreId = storeId

	if err := a.db.Save(&payment); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("update BTCPay invoice ID and store ID failed"), utils.ErrorBadRequest), nil)
		return
	}

	res := map[string]string{
		"invoiceID": string(invoiceRes.ID),
		"storeID":   storeId,
	}
	utils.ResponseOK(w, res)
}
