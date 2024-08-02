package webserver

import (
	"context"
	"fmt"
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/btcpay"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
)

type apiBTCPay struct {
	*WebServer
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
	//Get store ID from receiver ID
	storeId, storeErr := a.service.GetStoreIdFromUser(payment.SenderId)
	if storeErr != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(storeErr, utils.ErrorBadRequest), nil)
		return
	}
	invoiceRes, statusCode, err := a.btcpayClient.CreateInvoice(context.Background(), (*btcpay.StoreID)(&storeId), invoiceReq)

	if err != nil || statusCode != http.StatusOK {
		utils.Response(w, http.StatusBadRequest, utils.NewError(fmt.Errorf("Create invoice on BTCPay failed"), utils.ErrorBadRequest), nil)
		return
	}
	res := map[string]string{
		"invoiceID": string(invoiceRes.ID),
		"storeID":   storeId,
	}
	utils.ResponseOK(w, res)
}
