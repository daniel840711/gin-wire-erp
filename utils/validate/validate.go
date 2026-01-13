package validate

import (
	"encoding/json"
	"fmt"
	"interchange/internal/core"
	cErr "interchange/internal/pkg/error"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 輸出格式化的 validator error（欄位 json 名/型別/規則列表）
func ValidationErrorResponse(c *gin.Context, obj interface{}, err error) string {
	if errs, ok := err.(validator.ValidationErrors); ok {
		var b strings.Builder
		b.WriteString("Validation error:\n")
		for _, fe := range errs {
			field := jsonFieldName(obj, fe.StructField())
			ftype := fieldType(obj, fe.StructField())
			format := getFieldFormat(obj, fe.StructField())
			b.WriteString(fmt.Sprintf(" - Field \"%s\" (type: %s) failed the '%s' validation (rules: %v)\n",
				field, ftype, fe.Tag(), format))
		}
		return b.String()
	}
	return fmt.Sprintf("Validation error: %s", err.Error())
}

func jsonFieldName(obj interface{}, structField string) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if f, ok := t.FieldByName(structField); ok {
		tag := f.Tag.Get("json")
		if tag != "" && tag != "-" {
			return strings.Split(tag, ",")[0]
		}
	}
	return structField
}

func fieldType(obj interface{}, structField string) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if f, ok := t.FieldByName(structField); ok {
		return f.Type.Name()
	}
	return ""
}

func getFieldFormat(obj interface{}, structField string) []string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if f, ok := t.FieldByName(structField); ok {
		tag := f.Tag.Get("binding")
		if tag != "" {
			return strings.Split(tag, ",")
		}
	}
	return nil
}
func ParseObjectID(c *gin.Context, key string) (id primitive.ObjectID, cause error, responseErr error) {
	id, err := primitive.ObjectIDFromHex(c.Param(key))
	if err != nil {
		return primitive.NilObjectID, err, cErr.ValidatePathParamsErr("invalid " + key)
	}
	return id, nil, nil
}

func BindAndValidate(c *gin.Context, req any) (cause error, responseErr error) {
	if err := c.ShouldBindJSON(req); err != nil {
		return err, cErr.ValidateErr(ValidationErrorResponse(c, req, err))
	}
	return nil, nil
}
func GetInt64Query(c *gin.Context, key string, defaultVal int64) (int64, error) {
	if v := c.Query(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	}
	return defaultVal, nil
}
func PayloadToMap(payload any) (map[string]any, error) {
	// 先轉 JSON
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// 再轉回 map[string]any
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

var validProviders = []core.ProviderName{
	core.ProviderOpenAI,
	core.ProviderGemini,
	core.ProviderGrok,
	core.ProviderCustom,
}

func IsValidProviderName(provider string) bool {
	for _, v := range validProviders {
		if core.ProviderName(provider) == v {
			return true
		}
	}
	return false
}

// ===== ApiScope =====
var validApiScopes = []core.ApiScope{
	core.ApiScopeChatCompletions,
	core.ApiScopeImagesGenerations,
	core.ApiScopeImagesVariations,
	core.ApiScopeImagesEdits,
	core.ApiScopeAudioTranscriptions,
	core.ApiScopeEmbeddingsGenerations,
	core.ApiScopeGetModels,
}

func IsValidApiScope(scope string) bool {
	for _, v := range validApiScopes {
		if core.ApiScope(scope) == v {
			return true
		}
	}
	return false
}

// ===== Role =====
var validRoles = []core.Role{
	core.RoleAdmin,
	core.RoleEditor,
	core.RoleUser,
	core.RoleReadOnly,
	core.RoleBanned,
}

func IsValidRole(role string) bool {
	for _, v := range validRoles {
		if core.Role(role) == v {
			return true
		}
	}
	return false
}

// ===== Status =====
var validStatuses = []core.Status{
	core.StatusActive,
	core.StatusBlocked,
	core.StatusSuspended,
	core.StatusExpired,
	core.StatusRevoked,
	core.StatusMaintenance,
	core.StatusPending,
	core.StatusDeleted,
}

func IsValidStatus(status string) bool {
	for _, v := range validStatuses {
		if core.Status(status) == v {
			return true
		}
	}
	return false
}
