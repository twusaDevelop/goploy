package controller

import (
	"database/sql"
	"github.com/patrickmn/go-cache"
	"net/http"
	"strconv"
	"time"

	"github.com/zhenorzz/goploy/core"
	"github.com/zhenorzz/goploy/model"
)

// User struct
type User Controller

// Login -
func (User) Login(gp *core.Goploy) *core.Response {
	type ReqData struct {
		Account  string `json:"account" validate:"min=5,max=12"`
		Password string `json:"password" validate:"password"`
	}
	var reqData ReqData
	if err := verify(gp.Body, &reqData); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	userData, err := model.User{Account: reqData.Account}.GetDataByAccount()
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	if err := userData.Validate(reqData.Password); err != nil {
		return &core.Response{Code: core.Deny, Message: err.Error()}
	}

	if userData.State == model.Disable {
		return &core.Response{Code: core.AccountDisabled, Message: "Account is disabled"}
	}

	namespaceList, err := core.GetNamespace(userData.ID)
	if err != nil {
		return &core.Response{Code: core.Error, Message: "尚未分配空间，请联系管理员"}
	}

	token, err := userData.CreateToken()
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	model.User{ID: userData.ID, LastLoginTime: time.Now().Format("20060102150405")}.UpdateLastLoginTime()

	core.Cache.Set("userInfo:"+strconv.Itoa(int(userData.ID)), &userData, cache.DefaultExpiration)

	namespaceList, err = model.Namespace{UserID: userData.ID}.GetAllByUserID()
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	core.Cache.Set("namespace:"+strconv.Itoa(int(userData.ID)), &namespaceList, cache.DefaultExpiration)

	cookie := http.Cookie{Name: core.LoginCookieName, Value: token, Path: "/", MaxAge: 86400, HttpOnly: true}
	http.SetCookie(gp.ResponseWriter, &cookie)
	return &core.Response{
		Data: struct {
			Token         string           `json:"token"`
			NamespaceList model.Namespaces `json:"namespaceList"`
		}{Token: token, NamespaceList: namespaceList},
	}
}

// Info -
func (User) Info(gp *core.Goploy) *core.Response {
	type RespData struct {
		UserInfo struct {
			ID           int64  `json:"id"`
			Account      string `json:"account"`
			Name         string `json:"name"`
			SuperManager int64  `json:"superManager"`
		} `json:"userInfo"`
	}
	data := RespData{}
	data.UserInfo.ID = gp.UserInfo.ID
	data.UserInfo.Name = gp.UserInfo.Name
	data.UserInfo.Account = gp.UserInfo.Account
	data.UserInfo.SuperManager = gp.UserInfo.SuperManager
	return &core.Response{Data: data}
}

// GetList -
func (User) GetList(gp *core.Goploy) *core.Response {
	pagination, err := model.PaginationFrom(gp.URLQuery)
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	users, err := model.User{}.GetList(pagination)
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	return &core.Response{
		Data: struct {
			Users model.Users `json:"list"`
		}{Users: users},
	}
}

// GetTotal -
func (User) GetTotal(gp *core.Goploy) *core.Response {
	total, err := model.User{}.GetTotal()
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	return &core.Response{
		Data: struct {
			Total int64 `json:"total"`
		}{Total: total},
	}
}

// GetOption -
func (User) GetOption(gp *core.Goploy) *core.Response {
	users, err := model.User{}.GetAll()
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	return &core.Response{
		Data: struct {
			Users model.Users `json:"list"`
		}{Users: users},
	}
}

// Add user
func (User) Add(gp *core.Goploy) *core.Response {
	type ReqData struct {
		Account      string `json:"account" validate:"min=5,max=12"`
		Password     string `json:"password" validate:"omitempty,password"`
		Name         string `json:"name" validate:"required"`
		Contact      string `json:"contact" validate:"omitempty,len=11,numeric"`
		SuperManager int64  `json:"superManager" validate:"min=0,max=1"`
	}

	var reqData ReqData
	if err := verify(gp.Body, &reqData); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	userInfo, err := model.User{Account: reqData.Account}.GetDataByAccount()
	if err != nil && err != sql.ErrNoRows {
		return &core.Response{Code: core.Error, Message: err.Error()}
	} else if userInfo != (model.User{}) {
		return &core.Response{Code: core.Error, Message: "Account is already exist"}
	}
	id, err := model.User{
		Account:      reqData.Account,
		Password:     reqData.Password,
		Name:         reqData.Name,
		Contact:      reqData.Contact,
		SuperManager: reqData.SuperManager,
	}.AddRow()

	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	if reqData.SuperManager == model.SuperManager {
		if err := (model.NamespaceUser{UserID: id}).AddAdminByUserID(); err != nil {
			return &core.Response{Code: core.Error, Message: err.Error()}
		}
		if err := (model.ProjectUser{UserID: id}).AddAdminByUserID(); err != nil {
			return &core.Response{Code: core.Error, Message: err.Error()}
		}
	}

	return &core.Response{
		Data: struct {
			ID int64 `json:"id"`
		}{ID: id},
	}
}

// Edit user
func (User) Edit(gp *core.Goploy) *core.Response {
	type ReqData struct {
		ID           int64  `json:"id" validate:"gt=0"`
		Password     string `json:"password" validate:"omitempty,password"`
		Name         string `json:"name" validate:"required"`
		Contact      string `json:"contact" validate:"omitempty,len=11,numeric"`
		SuperManager int64  `json:"superManager" validate:"min=0,max=1"`
	}
	var reqData ReqData
	if err := verify(gp.Body, &reqData); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	userInfo, err := model.User{ID: reqData.ID}.GetData()

	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	err = model.User{
		ID:           reqData.ID,
		Password:     reqData.Password,
		Name:         reqData.Name,
		Contact:      reqData.Contact,
		SuperManager: reqData.SuperManager,
	}.EditRow()

	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	if userInfo.SuperManager == model.SuperManager && reqData.SuperManager == model.GeneralUser {
		if err := (model.NamespaceUser{UserID: reqData.ID}).DeleteByUserID(); err != nil {
			return &core.Response{Code: core.Error, Message: err.Error()}
		}
		if err := (model.ProjectUser{UserID: reqData.ID}).DeleteByUserID(); err != nil {
			return &core.Response{Code: core.Error, Message: err.Error()}
		}
	} else if userInfo.SuperManager == model.GeneralUser && reqData.SuperManager == model.SuperManager {
		if err := (model.NamespaceUser{UserID: reqData.ID}).AddAdminByUserID(); err != nil {
			return &core.Response{Code: core.Error, Message: err.Error()}
		}
	}

	return &core.Response{}
}

// RemoveRow User
func (User) Remove(gp *core.Goploy) *core.Response {
	type ReqData struct {
		ID int64 `json:"id" validate:"gt=0"`
	}
	var reqData ReqData
	if err := verify(gp.Body, &reqData); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	if reqData.ID == 1 {
		return &core.Response{Code: core.Error, Message: "Can not delete the super manager"}
	}
	if err := (model.User{ID: reqData.ID}).RemoveRow(); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	return &core.Response{}
}

// ChangePassword -
func (User) ChangePassword(gp *core.Goploy) *core.Response {
	type ReqData struct {
		OldPassword string `json:"oldPwd" validate:"password"`
		NewPassword string `json:"newPwd" validate:"password"`
	}
	var reqData ReqData
	if err := verify(gp.Body, &reqData); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	userData, err := model.User{ID: gp.UserInfo.ID}.GetData()
	if err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	if err := userData.Validate(reqData.OldPassword); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}

	if err := (model.User{ID: gp.UserInfo.ID, Password: reqData.NewPassword}).UpdatePassword(); err != nil {
		return &core.Response{Code: core.Error, Message: err.Error()}
	}
	return &core.Response{}
}
