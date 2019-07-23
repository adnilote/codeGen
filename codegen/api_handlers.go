package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	//"context"
	//"io/ioutil"
)

// CaseResponse
type CR map[string]interface{}
type Result struct {
	error    string
	response CR
}

func stringInSlice(a string, list []string) (bool, int) {
	for i, b := range list {
		if b == a {
			return true, i
		}
	}
	return false, -1
}

func (srv *MyApi) WrapperProfile(w http.ResponseWriter, r *http.Request) {

	var err error
	var Login string
	switch r.Method {
	case "GET":
		Login = r.URL.Query().Get(strings.ToLower("Login"))
	case "POST":
		Login = r.FormValue(strings.ToLower("Login"))
	}

	if Login == "" {
		writeRes(w,
			CR{
				"error": strings.ToLower("Login") + " must me not empty",
			},
			http.StatusBadRequest,
		)
		return
	}

	var in ProfileParams
	in = ProfileParams{
		Login: Login,
	}
	res, err := srv.Profile(r.Context(), in)
	if err != nil {
		_, ok := err.(ApiError)
		if ok {
			writeRes(w,
				CR{"error": err.(ApiError).Err.Error()},
				err.(ApiError).HTTPStatus,
			)
		} else {
			writeRes(w, CR{"error": "bad user"}, http.StatusInternalServerError)
		}
		return
	}
	writeRes(w, CR{"error": "", "response": res}, http.StatusOK)
	return

}
func (srv *MyApi) WrapperCreate(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		writeRes(w, CR{"error": "bad method"}, http.StatusNotAcceptable)
		return
	}
	if r.Header.Get("X-Auth") != "100500" {
		writeRes(w, CR{"error": "unauthorized"}, http.StatusForbidden)
		return
	}

	var err error
	var Login string
	Login = r.FormValue(strings.ToLower("Login"))

	if Login == "" {
		writeRes(w,
			CR{
				"error": strings.ToLower("Login") + " must me not empty",
			},
			http.StatusBadRequest,
		)
		return
	}

	if len(Login) < 10 {
		writeRes(w,
			CR{"error": strings.ToLower("Login") + " len must be >= 10"},
			http.StatusBadRequest,
		)
		return
	}

	var Name string
	Name = r.FormValue(strings.ToLower("full_name"))

	var Status string
	Status = r.FormValue(strings.ToLower("Status"))

	if Status == "" {
		Status = "user"
	}

	if ok, _ := stringInSlice(Status, strings.Split("user, moderator, admin", ", ")); !ok {
		writeRes(w,
			CR{
				"error": strings.ToLower("Status") + " must be one of [user, moderator, admin]",
			},
			http.StatusBadRequest,
		)
		return
	}

	var Age int
	Age, err = strconv.Atoi(r.FormValue(strings.ToLower("Age")))

	if err != nil {
		writeRes(w,
			CR{
				"error": strings.ToLower("Age") + " must be int",
			},
			http.StatusBadRequest,
		)
		return
	}

	if Age < 0 {
		writeRes(w,
			CR{"error": strings.ToLower("Age") + " must be >= 0"},
			http.StatusBadRequest,
		)
		return
	}

	if Age > 128 {
		writeRes(w,
			CR{"error": strings.ToLower("Age") + " must be <= 128"},
			http.StatusBadRequest,
		)
		return
	}

	var in CreateParams
	in = CreateParams{
		Age:    Age,
		Login:  Login,
		Name:   Name,
		Status: Status,
	}
	res, err := srv.Create(r.Context(), in)
	if err != nil {
		_, ok := err.(ApiError)
		if ok {
			writeRes(w,
				CR{"error": err.(ApiError).Err.Error()},
				err.(ApiError).HTTPStatus,
			)
		} else {
			writeRes(w, CR{"error": "bad user"}, http.StatusInternalServerError)
		}
		return
	}
	writeRes(w, CR{"error": "", "response": res}, http.StatusOK)
	return

}
func (srv *OtherApi) WrapperCreate(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		writeRes(w, CR{"error": "bad method"}, http.StatusNotAcceptable)
		return
	}
	if r.Header.Get("X-Auth") != "100500" {
		writeRes(w, CR{"error": "unauthorized"}, http.StatusForbidden)
		return
	}

	var err error
	var Username string
	Username = r.FormValue(strings.ToLower("Username"))

	if Username == "" {
		writeRes(w,
			CR{
				"error": strings.ToLower("Username") + " must me not empty",
			},
			http.StatusBadRequest,
		)
		return
	}

	if len(Username) < 3 {
		writeRes(
			w,
			CR{"error": strings.ToLower("Username") + " len must be >= 3"},
			http.StatusBadRequest,
		)
		return
	}

	var Name string
	Name = r.FormValue(strings.ToLower("account_name"))

	var Class string
	Class = r.FormValue(strings.ToLower("Class"))

	if Class == "" {
		Class = "warrior"
	}

	if ok, _ := stringInSlice(Class, strings.Split("warrior, sorcerer, rouge", ", ")); !ok {

		writeRes(w,
			CR{
				"error": strings.ToLower("Class") + " must be one of [warrior, sorcerer, rouge]",
			},
			http.StatusBadRequest,
		)
		return
	}

	Level, err := strconv.Atoi(r.FormValue(strings.ToLower("Level")))

	if err != nil {
		writeRes(w,
			CR{
				"error": strings.ToLower("Level") + " must be int",
			},
			http.StatusBadRequest,
		)
		return
	}

	if Level < 1 {
		writeRes(w,
			CR{"error": strings.ToLower("Level") + " must be >= 1"},
			http.StatusBadRequest,
		)
		return
	}

	if Level > 50 {
		writeRes(
			w,
			CR{"error": strings.ToLower("Level") + " must be <= 50"},
			http.StatusBadRequest,
		)
		return
	}

	var in OtherCreateParams
	in = OtherCreateParams{
		Username: Username,
		Name:     Name,
		Class:    Class,
		Level:    Level,
	}
	res, err := srv.Create(r.Context(), in)
	if err != nil {
		_, ok := err.(ApiError)
		if ok {
			writeRes(w,
				CR{"error": err.(ApiError).Err.Error()},
				err.(ApiError).HTTPStatus,
			)
		} else {
			writeRes(w, CR{"error": "bad user"}, http.StatusInternalServerError)
		}
		return
	}
	writeRes(w, CR{"error": "", "response": res}, http.StatusOK)
	return

}
func (this *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {

	case "/user/profile":
		this.WrapperProfile(w, r)

	case "/user/create":
		this.WrapperCreate(w, r)

	default:
		writeRes(w, CR{"error": "unknown method"}, http.StatusNotFound)
		return
	}
}

func (this *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {

	case "/user/create":
		this.WrapperCreate(w, r)

	default:
		writeRes(w, CR{"error": "unknown method"}, http.StatusNotFound)
		return
	}
}

func writeRes(w http.ResponseWriter, cr CR, status int) {
	a, _ := json.Marshal(cr)
	w.WriteHeader(status)
	w.Write(a)
}
