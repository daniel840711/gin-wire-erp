package repository

import (
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson"
)

// 統一管理所有 MySQL repository
type MongoDBRepository struct {
	userRepo                           *UserRepository
	userAPIKeyRepository               *UserAPIKeyRepository
	tenantRepository                   *TenantRepository
	organizationNodeRepository         *OrganizationNodeRepository
	storeRepository                    *StoreRepository
	employeeRepository                 *EmployeeRepository
	employeeOrganizationMembershipRepo *EmployeeOrganizationMembershipRepository
	employeeStoreMembershipRepo        *EmployeeStoreMembershipRepository
	permissionRepository               *PermissionRepository
	roleRepository                     *RoleRepository
	rolePermissionRepository           *RolePermissionRepository
	roleAssignmentRepository           *RoleAssignmentRepository
	permissionPolicyRepository         *PermissionPolicyRepository
	policyRuleRepository               *PolicyRuleRepository
	assignmentPolicyRepository         *AssignmentPolicyRepository
}

// 建立 MySQL repository 物件
func NewMongoDBRepository(
	userRepo *UserRepository,
	userAPIKeyRepository *UserAPIKeyRepository,
	tenantRepository *TenantRepository,
	organizationNodeRepository *OrganizationNodeRepository,
	storeRepository *StoreRepository,
	employeeRepository *EmployeeRepository,
	employeeOrganizationMembershipRepo *EmployeeOrganizationMembershipRepository,
	employeeStoreMembershipRepo *EmployeeStoreMembershipRepository,
	permissionRepository *PermissionRepository,
	roleRepository *RoleRepository,
	rolePermissionRepository *RolePermissionRepository,
	roleAssignmentRepository *RoleAssignmentRepository,
	permissionPolicyRepository *PermissionPolicyRepository,
	policyRuleRepository *PolicyRuleRepository,
	assignmentPolicyRepository *AssignmentPolicyRepository,
) *MongoDBRepository {
	return &MongoDBRepository{
		userRepo:                           userRepo,
		userAPIKeyRepository:               userAPIKeyRepository,
		tenantRepository:                   tenantRepository,
		organizationNodeRepository:         organizationNodeRepository,
		storeRepository:                    storeRepository,
		employeeRepository:                 employeeRepository,
		employeeOrganizationMembershipRepo: employeeOrganizationMembershipRepo,
		employeeStoreMembershipRepo:        employeeStoreMembershipRepo,
		permissionRepository:               permissionRepository,
		roleRepository:                     roleRepository,
		rolePermissionRepository:           rolePermissionRepository,
		roleAssignmentRepository:           roleAssignmentRepository,
		permissionPolicyRepository:         permissionPolicyRepository,
		policyRuleRepository:               policyRuleRepository,
		assignmentPolicyRepository:         assignmentPolicyRepository,
	}
}

// Wire 依賴提供
var ProviderSet = wire.NewSet(
	NewUserRepository,
	NewUserAPIKeyRepository,
	NewTenantRepository,
	NewOrganizationNodeRepository,
	NewStoreRepository,
	NewEmployeeRepository,
	NewEmployeeOrganizationMembershipRepository,
	NewEmployeeStoreMembershipRepository,
	NewPermissionRepository,
	NewRoleRepository,
	NewRolePermissionRepository,
	NewRoleAssignmentRepository,
	NewPermissionPolicyRepository,
	NewPolicyRuleRepository,
	NewAssignmentPolicyRepository,
	NewMongoDBRepository)

func withUpdatedAt(update bson.M) bson.M {
	// 確保 $currentDate 存在
	currentDate, ok := update["$currentDate"].(bson.M)
	if !ok || currentDate == nil {
		currentDate = bson.M{}
	}
	currentDate["updatedAt"] = true
	update["$currentDate"] = currentDate
	return update
}
