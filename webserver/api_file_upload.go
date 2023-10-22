package webserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/utils"
)

type apiFileUpload struct {
	*WebServer
}

var imagePath = getBinPath() + "\\upload\\product-image"

func (a *apiFileUpload) uploadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	var fileNumber = len(r.MultipartForm.File)
	var newImagesName = r.Form.Get("newImagesName")
	var newImageNameArr = strings.Split(newImagesName, ";")
	for i := 0; i < fileNumber; i++ {
		file, handler, err := r.FormFile("files[" + strconv.Itoa(i) + "]")
		if err != nil {
			fmt.Println("Error Retrieving the File")
			fmt.Println(err)
			return
		}
		defer file.Close()
		err = os.MkdirAll(imagePath, os.ModePerm)
		if err != nil {
			fmt.Println("Create folder failed")
			return
		}
		var fileNameArr = strings.Split(handler.Filename, ".")
		if len(fileNameArr) < 2 {
			fmt.Println("File error")
			continue
		}
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println(err)
		}
		err = ioutil.WriteFile(imagePath+"\\"+newImageNameArr[i], fileBytes, 0777)
		if err != nil {
			fmt.Println("Write file error")
		}
	}
}

func (a *apiFileUpload) getProductImagesBase64(w http.ResponseWriter, r *http.Request) {
	var avatar = r.FormValue("avatar")
	var images = r.FormValue("images")
	var base64Map = Map{}
	if !utils.IsEmpty(avatar) {
		base64Map[avatar] = utils.ConvertImageToBase64(avatar)
	}
	if !utils.IsEmpty(images) {
		var galleryArr = strings.Split(images, ",")
		for _, image := range galleryArr {
			base64Map[image] = utils.ConvertImageToBase64(image)
		}
	}
	utils.ResponseOK(w, base64Map)
}

func (a *apiFileUpload) getImageBase64(w http.ResponseWriter, r *http.Request) {
	var imageNames = r.FormValue("imageNames")
	if utils.IsEmpty(imageNames) {
		return
	}
	var base64Map = Map{}
	var galleryArr = strings.Split(imageNames, ",")
	for _, image := range galleryArr {
		base64Map[image] = utils.ConvertImageToBase64(image)
	}
	utils.ResponseOK(w, base64Map)
}

func getBinPath() string {
	e, err := os.Executable()
	if err != nil {
		panic(err)
	}
	path := path.Dir(e)
	return path
}
