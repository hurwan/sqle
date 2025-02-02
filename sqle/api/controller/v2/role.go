package v2

import (
	"net/http"

	"github.com/actiontech/sqle/sqle/api/controller"
	"github.com/actiontech/sqle/sqle/model"

	"github.com/labstack/echo/v4"
)

type CreateRoleReqV2 struct {
	Name           string   `json:"role_name" form:"role_name" valid:"required,name"`
	Desc           string   `json:"role_desc" form:"role_desc"`
	Instances      []string `json:"instance_name_list" form:"instance_name_list"`
	OperationCodes []uint   `json:"operation_code_list" form:"operation_code_list"`
	Users          []string `json:"user_name_list,omitempty" form:"user_name_list"`
	UserGroups     []string `json:"user_group_name_list,omitempty" form:"user_group_name_list"`
}

// @Summary 创建角色
// @Description create role
// @Id createRoleV2
// @Tags role
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param instance body v2.CreateRoleReqV2 true "create role"
// @Success 200 {object} controller.BaseRes
// @router /v2/roles [post]
func CreateRole(c echo.Context) (err error) {

	req := new(CreateRoleReqV2)
	{
		if err := controller.BindAndValidateReq(c, req); err != nil {
			return err
		}
	}

	s := model.GetStorage()

	// check if role name already exists
	{
		_, exist, err := s.GetRoleByName(req.Name)
		if err != nil {
			return controller.JSONBaseErrorReq(c, err)
		}
		if exist {
			return controller.JSONNewDataExistErr(c, "role<%s> is exist", req.Name)
		}
	}

	// check instances
	var instances []*model.Instance
	{
		if len(req.Instances) > 0 {
			instances, err = s.GetAndCheckInstanceExist(req.Instances)
			if err != nil {
				return controller.JSONBaseErrorReq(c, err)
			}
		}
	}

	// check operation codes
	{
		if len(req.OperationCodes) > 0 {
			if err := model.CheckIfOperationCodeValid(req.OperationCodes); err != nil {
				return controller.JSONBaseErrorReq(c, err)
			}
		}
	}

	// check users
	var users []*model.User
	{
		if len(req.Users) > 0 {
			users, err = s.GetAndCheckUserExist(req.Users)
			if err != nil {
				return controller.JSONBaseErrorReq(c, err)
			}
		}
	}

	// check user groups
	var userGroups []*model.UserGroup
	{
		if len(req.UserGroups) > 0 {
			userGroups, err = s.GetAndCheckUserGroupExist(req.UserGroups)
			if err != nil {
				return controller.JSONBaseErrorReq(c, err)
			}
		}
	}

	newRole := &model.Role{
		Name: req.Name,
		Desc: req.Desc,
	}

	return controller.JSONBaseErrorReq(c,
		s.SaveRoleAndAssociations(newRole, instances, req.OperationCodes, users, userGroups),
	)
}

type GetRolesReqV2 struct {
	FilterRoleName     string `json:"filter_role_name" query:"filter_role_name"`
	FilterUserName     string `json:"filter_user_name" query:"filter_user_name"`
	FilterInstanceName string `json:"filter_instance_name" query:"filter_instance_name"`
	PageIndex          uint32 `json:"page_index" query:"page_index" valid:"required"`
	PageSize           uint32 `json:"page_size" query:"page_size" valid:"required"`
}

type GetRolesResV2 struct {
	controller.BaseRes
	Data      []*RoleResV2 `json:"data"`
	TotalNums uint64       `json:"total_nums"`
}

type Operation struct {
	Code uint   `json:"op_code"`
	Desc string `json:"op_desc"`
}

type RoleResV2 struct {
	Name       string       `json:"role_name"`
	Desc       string       `json:"role_desc"`
	Users      []string     `json:"user_name_list,omitempty"`
	Instances  []string     `json:"instance_name_list,omitempty"`
	Operations []*Operation `json:"operation_list,omitempty"`
	UserGroups []string     `json:"user_group_name_list,omitempty" form:"user_group_name_list"`
	IsDisabled bool         `json:"is_disabled,omitempty"`
}

// @Summary 获取角色列表
// @Description get role list
// @Id getRoleListV2
// @Tags role
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param filter_role_name query string false "filter role name"
// @Param filter_user_name query string false "filter user name"
// @Param filter_instance_name query string false "filter instance name"
// @Param page_index query uint32 false "page index"
// @Param page_size query uint32 false "size of per page"
// @Success 200 {object} v2.GetRolesResV2
// @router /v2/roles [get]
func GetRoles(c echo.Context) error {
	req := new(GetRolesReqV2)
	{
		if err := controller.BindAndValidateReq(c, req); err != nil {
			return err
		}
	}

	s := model.GetStorage()

	var queryCondition map[string]interface{}
	{
		limit, offset := controller.GetLimitAndOffset(req.PageIndex, req.PageSize)
		queryCondition = map[string]interface{}{
			"filter_role_name":     req.FilterRoleName,
			"filter_user_name":     req.FilterUserName,
			"filter_instance_name": req.FilterInstanceName,
			"limit":                limit,
			"offset":               offset,
		}
	}

	roles, count, err := s.GetRolesByReq(queryCondition)
	if err != nil {
		return controller.JSONBaseErrorReq(c, err)
	}

	roleRes := make([]*RoleResV2, len(roles))
	for i := range roles {
		ops := make([]*Operation, len(roles[i].OperationsCodes))
		opCodes := roles[i].OperationsCodes.ForceConvertIntSlice()
		for i := range opCodes {
			ops[i] = &Operation{
				Code: opCodes[i],
				Desc: model.GetOperationCodeDesc(opCodes[i]),
			}
		}
		roleRes[i] = &RoleResV2{
			Name:       roles[i].Name,
			Desc:       roles[i].Desc,
			Instances:  roles[i].InstanceNames,
			UserGroups: roles[i].UserGroupNames,
			Users:      roles[i].UserNames,
			IsDisabled: roles[i].IsDisabled(),
			Operations: ops,
		}

	}

	return c.JSON(http.StatusOK, &GetRolesResV2{
		BaseRes:   controller.NewBaseReq(nil),
		Data:      roleRes,
		TotalNums: count,
	})
}

type UpdateRoleReqV2 struct {
	Desc           *string   `json:"role_desc" form:"role_desc"`
	Users          *[]string `json:"user_name_list,omitempty" form:"user_name_list"`
	Instances      *[]string `json:"instance_name_list,omitempty" form:"instance_name_list"`
	OperationCodes *[]uint   `json:"operation_code_list,omitempty" form:"operation_code_list"`
	UserGroups     *[]string `json:"user_group_name_list,omitempty" form:"user_group_name_list"`
	IsDisabled     *bool     `json:"is_disabled,omitempty"`
}

// @Summary 更新角色信息
// @Description update role
// @Id updateRoleV2
// @Tags role
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param role_name path string true "role name"
// @Param instance body v2.UpdateRoleReqV2 true "update role request"
// @Success 200 {object} controller.BaseRes
// @router /v2/roles/{role_name}/ [patch]
func UpdateRole(c echo.Context) (err error) {

	req := new(UpdateRoleReqV2)
	{
		if err := controller.BindAndValidateReq(c, req); err != nil {
			return err
		}
	}

	s := model.GetStorage()
	roleName := c.Param("role_name")

	// check if role name exists
	var role *model.Role
	{
		var isExist bool
		role, isExist, err = s.GetRoleByName(roleName)
		if err != nil {
			if err != nil {
				return controller.JSONBaseErrorReq(c, err)
			}
		}
		if !isExist {
			return controller.JSONNewDataNotExistErr(c,
				`role is not exist`)
		}
	}

	// update stat
	{
		if req.IsDisabled != nil {
			if *req.IsDisabled {
				role.Stat = model.Disabled
			} else {
				role.Stat = model.Enabled
			}

		}
	}

	// update desc
	if req.Desc != nil {
		role.Desc = *req.Desc
	}

	// check instances
	var instances []*model.Instance
	{
		if req.Instances != nil {
			if len(*req.Instances) > 0 {
				instances, err = s.GetAndCheckInstanceExist(*req.Instances)
				if err != nil {
					return controller.JSONBaseErrorReq(c, err)
				}
			} else {
				instances = []*model.Instance{}
			}
		}
	}

	// check operation codes
	var opCodes []uint
	{
		if req.OperationCodes != nil {
			if len(*req.OperationCodes) > 0 {
				if err := model.CheckIfOperationCodeValid(*req.OperationCodes); err != nil {
					return controller.JSONBaseErrorReq(c, err)
				}
				opCodes = *req.OperationCodes
			} else {
				opCodes = make([]uint, 0)
			}
		}
	}

	// check users
	var users []*model.User
	{
		if req.Users != nil {
			if len(*req.Users) > 0 {
				users, err = s.GetAndCheckUserExist(*req.Users)
				if err != nil {
					return controller.JSONBaseErrorReq(c, err)
				}
			} else {
				users = make([]*model.User, 0)
			}
		}
	}

	// check user groups
	var userGroups []*model.UserGroup
	{
		if req.UserGroups != nil {
			if len(*req.UserGroups) > 0 {
				userGroups, err = s.GetAndCheckUserGroupExist(*req.UserGroups)
				if err != nil {
					return controller.JSONBaseErrorReq(c, err)
				}
			} else {
				userGroups = make([]*model.UserGroup, 0)
			}
		}
	}

	return controller.JSONBaseErrorReq(c,
		s.SaveRoleAndAssociations(role, instances, opCodes, users, userGroups),
	)

}
