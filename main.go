package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"system/entity"
	"system/service"
)

var ser = service.Connection{}

const adminTokenId = "Admin001"

func init() {
	ser.Server = "mongodb://localhost:27017"
	ser.Database = "Dummy"
	ser.Collection = "idData"

	ser.Connect()
}

func storeData(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if token := r.Header.Get("tokenid"); token != adminTokenId {
		respondWithError(w, http.StatusBadRequest, "Unauthorized")
		return
	}

	if r.Method != "POST" {
		respondWithError(w, http.StatusBadRequest, "Invalid Method")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "The uploaded file is too big. Please choose an file that's less than 1MB in size", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["file"]
	reqBody := r.MultipartForm.Value["request"][0]
	if len(files) > 1 {
		respondWithError(w, http.StatusBadRequest, "Please upload only One file")
		return
	}

	var dataBody entity.Request

	if err := json.Unmarshal([]byte(reqBody), &dataBody); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Request")
		return
	}

	if result, err := ser.CreateIdAndStore(dataBody, files); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
	} else {
		respondWithJson(w, http.StatusBadRequest, result)
	}
}

func search(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "GET" {
		respondWithError(w, http.StatusBadRequest, "Invalid Method")
		return
	}

	if result, err := ser.FetchAllData(); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
	} else {
		respondWithJson(w, http.StatusBadRequest, result)
	}
}

func fetchDataByIdCard(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "GET" {
		respondWithError(w, http.StatusBadRequest, "Invalid Method")
		return
	}

	segment := strings.Split(r.URL.Path, "/")
	cardId := segment[len(segment)-1]
	if cardId == "" {
		respondWithError(w, http.StatusBadRequest, "Please provide Id for Search")
	}

	if result, err := ser.FetchDataByIdCard(cardId); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
	} else {
		respondWithJson(w, http.StatusBadRequest, result)
	}
}

func update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if token := r.Header.Get("tokenid"); token != adminTokenId {
		respondWithError(w, http.StatusBadRequest, "Unauthorized")
		return
	}

	if r.Method != "PUT" {
		respondWithError(w, http.StatusBadRequest, "Invalid Method")
		return
	}
	path := r.URL.Path
	segments := strings.Split(path, "/")
	id := segments[len(segments)-1]
	var dataBody entity.Request
	if err := json.NewDecoder(r.Body).Decode(&dataBody); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Request")
		return
	}

	if result, err := ser.UpdateDataById(id, dataBody); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
	} else {
		respondWithJson(w, http.StatusBadRequest, result)
	}
}

func delete(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if token := r.Header.Get("tokenid"); token != adminTokenId {
		respondWithError(w, http.StatusBadRequest, "Unauthorized")
		return
	}

	if r.Method != "DELETE" {
		respondWithError(w, http.StatusBadRequest, "Invalid Method")
		return
	}
	path := r.URL.Path
	segments := strings.Split(path, "/")
	id := segments[len(segments)-1]

	if result, err := ser.DeleteById(id); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
	} else {
		respondWithJson(w, http.StatusBadRequest, result)
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJson(w, code, map[string]string{"error": msg})
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	http.HandleFunc("/create-id", storeData)
	http.HandleFunc("/search", search)
	http.HandleFunc("/search-by-id/", fetchDataByIdCard)
	http.HandleFunc("/update-by-id/", update)
	http.HandleFunc("/delete-by-id/", delete)
	log.Println("Server started at 8080")
	http.ListenAndServe(":8080", nil)
}
