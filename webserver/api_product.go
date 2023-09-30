package webserver

import (
	"fmt"
	"net/http"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"code.cryptopower.dev/mgmt-ng/be/webserver/portal"
	"github.com/go-chi/chi/v5"
)

type apiProduct struct {
	*WebServer
}

func (a *apiProduct) info(w http.ResponseWriter, r *http.Request) {
	var id = chi.URLParam(r, "id")
	product, err := a.service.GetProductInfo(utils.Uint64(id))
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	} else {
		utils.ResponseOK(w, product)
	}
}

func (a *apiProduct) updateProduct(w http.ResponseWriter, r *http.Request) {
	var body portal.UpdateProductRequest
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		utils.Response(w, http.StatusBadRequest, err, nil)
		return
	}

	product, err := a.service.UpdateProduct(body.Id, body)
	if err != nil {
		utils.Response(w, http.StatusInternalServerError, err, nil)
		return
	}

	utils.ResponseOK(w, product)
}

// Get product for seller
func (a *apiProduct) getListProducts(w http.ResponseWriter, r *http.Request) {
	var f storage.ProductFilter
	if err := a.parseQueryAndValidate(r, &f); err != nil {
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	// if utils.IsEmpty(f.Sort.Order) {
	// 	f.Sort.Order = "lastSeen desc"
	// }
	var products []storage.Product
	if err := a.db.GetList(&f, &products); err != nil {
		utils.Response(w, http.StatusInternalServerError, utils.NewError(err, utils.ErrorInternalCode), nil)
		return
	}
	count, _ := a.db.Count(&f, &storage.Product{})
	utils.ResponseOK(w, Map{
		"products": products,
		"count":    count,
	})
}

func (a *apiProduct) createProduct(w http.ResponseWriter, r *http.Request) {
	var body portal.CreateProductForm
	err := a.parseJSONAndValidate(r, &body)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusBadRequest, utils.NewError(err, utils.ErrorBadRequest), nil)
		return
	}
	userInfo, _ := a.credentialsInfo(r)

	var ownerName = userInfo.DisplayName
	if utils.IsEmpty(ownerName) {
		ownerName = userInfo.UserName
	}
	product, err := a.service.CreateProduct(userInfo.Id, ownerName, body)
	if err != nil {
		log.Error(err)
		utils.Response(w, http.StatusOK, err, nil)
		return
	}
	res := Map{
		"product": product,
	}
	utils.ResponseOK(w, res, nil)
}

func (a *apiProduct) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	product, err := a.service.GetProductInfo(utils.Uint64(id))
	if err != nil {
		utils.Response(w, http.StatusNotFound, err, nil)
	}
	product.Status = uint32(utils.Deleted)

	pr, err := a.service.UpdateSingleProduct(product)
	if err != nil {
		fmt.Println("Delete failed")
	}
	utils.ResponseOK(w, pr, nil)
}
