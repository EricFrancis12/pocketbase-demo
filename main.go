package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type User struct {
	Id              string `db:"id" json:"id"`
	Email           string `db:"email" json:"email"`
	EmailVisibility bool   `db:"emailVisibility" json:"emailVisibility"`
	Verified        bool   `db:"verified" json:"verified"`
	Name            string `db:"name" json:"name"`
	Avatar          string `db:"avatar" json:"avatar"`
	Created         string `db:"created" json:"created"`
	Updated         string `db:"updated" json:"updated"`
}

type UserCreationRequest struct {
	Email           string `db:"email" json:"email"`
	EmailVisibility bool   `db:"emailVisibility" json:"emailVisibility"`
	Name            string `db:"name" json:"name"`
}

type UserUpdateRequest struct {
	Email           *string `db:"email" json:"email"`
	EmailVisibility *bool   `db:"emailVisibility" json:"emailVisibility"`
	Name            *string `db:"name" json:"name"`
}

type Storage struct {
}

type APIResp struct {
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func NewAPIResp(success bool, message string, data any) *APIResp {
	return &APIResp{
		Success: success,
		Message: message,
		Data:    data,
	}
}

func WriteOK(e *core.RequestEvent, message string, data any) error {
	return e.JSON(http.StatusOK, NewAPIResp(true, message, data))
}

func WriteBadRequest(e *core.RequestEvent, message string, data any) error {
	return e.JSON(http.StatusBadRequest, NewAPIResp(false, message, data))
}

func WriteInternalServerError(e *core.RequestEvent, message string, data any) error {
	return e.JSON(http.StatusInternalServerError, NewAPIResp(false, message, data))
}

func GetUsers(app *pocketbase.PocketBase) ([]User, error) {
	users := []User{}
	err := app.DB().
		NewQuery("SELECT * FROM users").
		All(&users)
	if err != nil {
		return []User{}, err
	}
	return users, nil
}

func GetUserById(app *pocketbase.PocketBase, userId string) (*User, error) {
	user := User{}
	err := app.DB().
		NewQuery("SELECT * FROM users WHERE id={:userId}").
		Bind(dbx.Params{
			"userId": userId,
		}).
		One(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByEmail(app *pocketbase.PocketBase, email string) (*User, error) {
	user := User{}
	err := app.DB().
		NewQuery("SELECT * FROM users WHERE email={:email}").
		Bind(dbx.Params{
			"email": email,
		}).
		One(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func InsertUser(app *pocketbase.PocketBase, cr UserCreationRequest) (*User, error) {
	_, err := app.DB().
		NewQuery("INSERT INTO users (email, emailVisibility, name) VALUES ({:email}, {:emailVisibility}, {:name})").
		Bind(dbx.Params{
			"email":           cr.Email,
			"emailVisibility": cr.EmailVisibility,
			"name":            cr.Name,
		}).
		Execute()
	if err != nil {
		return nil, err
	}
	return GetUserByEmail(app, cr.Email)
}

func UpdateUserById(app *pocketbase.PocketBase, userId string, ur UserUpdateRequest) (*User, error) {
	values := []string{}
	params := dbx.Params{
		"userId": userId,
	}
	if ur.Email != nil {
		values = append(values, "email={:email}")
		params["email"] = *ur.Email
	}
	if ur.EmailVisibility != nil {
		values = append(values, "emailVisibility={:emailVisibility}")
		params["emailVisibility"] = *ur.EmailVisibility
	}
	if ur.Name != nil {
		values = append(values, "name={:name}")
		params["name"] = *ur.Name
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("empty update request")
	}
	query := fmt.Sprintf("UPDATE users SET %s WHERE id={:userId}", strings.Join(values, ", "))
	_, err := app.DB().
		NewQuery(query).
		Bind(params).
		Execute()
	if err != nil {
		return nil, err
	}
	return GetUserById(app, userId)
}

func DeleteUserById(app *pocketbase.PocketBase, userId string) error {
	_, err := app.DB().
		NewQuery("DELETE FROM users WHERE id={:userId}").
		Bind(dbx.Params{
			"userId": userId,
		}).
		Execute()
	return err
}

func HandleGetUsers(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		users, err := GetUsers(app)
		if err != nil {
			return WriteInternalServerError(e, "error getting users: "+err.Error(), nil)
		}
		return WriteOK(e, "", users)
	}
}

func HandleGetUserById(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		userId := e.Request.PathValue("userId")
		user, err := GetUserById(app, userId)
		if err != nil {
			return WriteInternalServerError(e, "error getting user: "+err.Error(), nil)
		}
		return WriteOK(e, "", user)
	}
}

func HandleInsertUser(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		cr := UserCreationRequest{}
		if err := e.BindBody(&cr); err != nil {
			return WriteBadRequest(e, "bad request: "+err.Error(), nil)
		}
		user, err := InsertUser(app, cr)
		if err != nil {
			return WriteInternalServerError(e, "error creating new user: "+err.Error(), nil)
		}
		return WriteOK(e, "", user)
	}
}

func HandleUpdateUserById(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		userId := e.Request.PathValue("userId")
		ur := UserUpdateRequest{}
		if err := e.BindBody(&ur); err != nil {
			return WriteBadRequest(e, "bad request: "+err.Error(), nil)
		}
		_, err := UpdateUserById(app, userId, ur)
		if err != nil {
			return WriteInternalServerError(e, "error creating new user: "+err.Error(), nil)
		}
		return WriteOK(e, "", nil)
	}
}

func HandleDeleteUserById(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		userId := e.Request.PathValue("userId")
		err := DeleteUserById(app, userId)
		if err != nil {
			return WriteInternalServerError(e, "error deleting user: "+err.Error(), nil)
		}
		return WriteOK(e, "", nil)
	}
}

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/users", HandleGetUsers(app))
		se.Router.GET("/users/{userId}", HandleGetUserById(app))
		se.Router.POST("/users", HandleInsertUser(app))
		se.Router.PATCH("/users/{userId}", HandleUpdateUserById(app))
		se.Router.DELETE("/users/{userId}", HandleDeleteUserById(app))

		// serves static files from the provided public dir (if exists)
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), false))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
