# Schema 規劃與 RBAC 說明書

適用 MongoDB / SQL 的多租戶組織與權限設計規劃。本文檔提供：資料表/集合欄位、索引建議、權限計算規則，以及 Repository/Service/Controller 的設計藍圖。

## 1. 使用範圍與目標

- 適用場景：多租戶 SaaS、組織樹與門店結構、後台 RBAC 權限控制。
- 目標：可清楚落地至 MongoDB 或 SQL，並便於開發與後續擴充。
- 原則：結構清晰、索引可追溯、權限計算可解釋、接口有一致性。

## 2. 命名與欄位通用約定

| 項目 | 約定 |
| --- | --- |
| id | string（UUID/ObjectId）主鍵，必填 |
| tenantId | string，租戶範圍欄位；`permissions` 可允許 null 代表系統內建 |
| createdAt / updatedAt | datetime，必填（除非另標註可選） |
| status | string/enum，建議使用 `active/inactive/archived` |
| 軟刪除 | 不直接刪除資料，改以 status 表示 |

額外約定：

- 組織/門店樹：以 `parentId` + `path`（`/root/child/...`）表示層級與子樹。
- 欄位命名：採 `camelCase`，跨表關聯用 `xxxId`。
- 索引命名：以欄位名組合表示，如 `(tenantId, roleId)`。

## 3. 實體總覽（目錄）

- A. 租戶與組織圖
  - tenants
  - organization_nodes
- B. 店與組織歸屬
  - stores
- C. 員工/使用者
  - employees
  - employee_organization_memberships
  - employee_store_memberships
- D. 權限定義
  - permissions
- E. 角色
  - roles
  - role_permissions
- F. 指派與範圍
  - role_assignments
- G. Data Policy（資料範圍）
  - permission_policies
  - policy_rules
  - assignment_policies
- H. 組織關係與權限計算（規則說明）
- I. Repository 設計藍圖
- J. Service 組合與責任
- K. Controller / API 範例

## A. 租戶與組織圖

### tenants

用途：租戶/客戶主體資訊。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| name | string | Y | 租戶名稱 | UNIQUE(建議) |
| status | string/enum | Y | 狀態（例：active/inactive） | IDX(建議) |
| createdAt | datetime | Y | 建立時間 | — |
| updatedAt | datetime | Y | 更新時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (name) | Y | 防重複租戶名稱（按需求） |

### organization_nodes

用途：組織樹（company/division/department/team 等）。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, parentId), IDX(tenantId, path) |
| name | string | Y | 節點名稱 | — |
| type | string/enum | Y | company/division/department/team（可擴充） | — |
| parentId | string | N | 父節點，root 為 null | IDX(tenantId, parentId) |
| path | string | Y | 例：/rootId/divId/deptId | IDX(tenantId, path) |
| depth | int | Y | 深度 | — |
| managerEmployeeId | string | N | 負責人 employeeId | IDX(建議) |
| status | string/enum | Y | 狀態 | IDX(建議) |
| createdAt | datetime | Y | 建立時間 | — |
| updatedAt | datetime | Y | 更新時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, parentId) | N | 找同層子節點 |
| (tenantId, path) | N | 前綴查詢子樹 |
| (tenantId, managerEmployeeId) | N | 負責人查詢（可選） |

## B. 店與組織歸屬

### stores

用途：門店與組織節點歸屬。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, code) |
| code | string | Y | 門店編碼 | UNIQUE(tenantId, code) |
| name | string | Y | 門店名稱 | — |
| organizationNodeId | string | Y | 所屬組織節點 | IDX(建議) |
| region | string | N | 區域（可選） | — |
| status | string/enum | Y | 狀態 | IDX(建議) |
| createdAt | datetime | Y | 建立時間 | — |
| updatedAt | datetime | Y | 更新時間 | — |
| parentStoreId | string | N | 總店/分店樹（可選） | IDX(建議) |
| path | string | N | 門店樹路徑（可選） | IDX(建議) |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, code) | Y | 門店編碼唯一 |
| (tenantId, organizationNodeId) | N | 按組織查門店 |
| (tenantId, parentStoreId) | N | 門店樹（可選） |
| (tenantId, path) | N | 門店子樹（可選） |

## C. 員工/使用者

### employees

用途：ERP 後台帳號（Employee = 後台帳號）。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, account) |
| account | string | Y | 登入帳號/信箱/手機 | UNIQUE(tenantId, account) |
| passwordHash | string | Y | 密碼雜湊或外部 SSO | — |
| displayName | string | Y | 顯示名稱 | — |
| status | string/enum | Y | active/suspended/resigned | IDX(建議) |
| primaryOrganizationNodeId | string | Y | 主要部門 | IDX(建議) |
| primaryStoreId | string | N | 主要門店 | IDX(建議) |
| jobTitle | string | N | 職稱 | — |
| reportToEmployeeId | string | N | 直屬主管 | IDX(建議) |
| createdAt | datetime | Y | 建立時間 | — |
| updatedAt | datetime | Y | 更新時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, account) | Y | 帳號唯一 |
| (tenantId, primaryOrganizationNodeId) | N | 按部門找人 |
| (tenantId, primaryStoreId) | N | 按門店找人 |
| (tenantId, reportToEmployeeId) | N | 匯報樹查詢 |

### employee_organization_memberships

用途：員工多部門兼任。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, employeeId) |
| employeeId | string | Y | 員工 | IDX(tenantId, employeeId) |
| organizationNodeId | string | Y | 部門/組織節點 | IDX(建議) |
| roleInOrganization | string/enum | N | member/manager | — |
| createdAt | datetime | Y | 建立時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, employeeId) | N | 員工歸屬查詢 |
| (tenantId, organizationNodeId) | N | 部門成員查詢 |
| (tenantId, employeeId, organizationNodeId) | Y | 防止重複兼任 |

### employee_store_memberships

用途：員工多門店兼任。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, employeeId) |
| employeeId | string | Y | 員工 | IDX(tenantId, employeeId) |
| storeId | string | Y | 門店 | IDX(建議) |
| position | string/enum | N | clerk/leader/manager | — |
| createdAt | datetime | Y | 建立時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, employeeId) | N | 員工歸屬查詢 |
| (tenantId, storeId) | N | 門店成員查詢 |
| (tenantId, employeeId, storeId) | Y | 防止重複兼任 |

## D. 權限定義

### permissions

用途：權限定義，建議用「資源 + 動作」。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | N | null=系統內建，非 null=租戶自訂 | IDX(tenantId, key) |
| key | string | Y | 例：orders.read | UNIQUE(tenantId, key) |
| resource | string | Y | 例：orders | IDX(建議) |
| action | string/enum | Y | read/create/update/delete/approve/export... | IDX(建議) |
| description | string | N | 說明 | — |
| status | string/enum | Y | 狀態 | IDX(建議) |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, key) | Y | 權限鍵唯一 |
| (resource, action) | N | 權限篩選 |

## E. 角色

### roles

用途：權限集合。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, name) |
| name | string | Y | 例：店長/倉管/客服/財務 | IDX(tenantId, name) |
| code | string | N | 角色代碼 | UNIQUE(建議) |
| isSystem | bool | Y | 系統預設不可刪 | — |
| status | string/enum | Y | 狀態 | IDX(建議) |
| createdAt | datetime | Y | 建立時間 | — |
| updatedAt | datetime | Y | 更新時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, name) | Y | 角色名唯一（按需求） |
| (tenantId, code) | Y | 角色代碼唯一（按需求） |

### role_permissions

用途：角色與權限的多對多關聯。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, roleId) |
| roleId | string | Y | 角色 | IDX(tenantId, roleId) |
| permissionId | string | Y | 權限 | IDX(建議) |
| createdAt | datetime | Y | 建立時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, roleId) | N | 角色權限查詢 |
| (tenantId, permissionId) | N | 權限反查 |
| (tenantId, roleId, permissionId) | Y | 防止重複綁定 |

## F. 指派與範圍

### role_assignments

用途：把角色指派給主體，並定義範圍（scope）。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, subjectType, subjectId) |
| subjectType | string/enum | Y | employee（可擴充 group/apiKey） | IDX(tenantId, subjectType, subjectId) |
| subjectId | string | Y | employeeId | IDX(tenantId, subjectType, subjectId) |
| roleId | string | Y | 角色 | IDX(tenantId, roleId) |
| scopeType | string/enum | Y | tenant/organizationNode/store/organizationSubtree/storeGroup | IDX(建議) |
| scopeId | string | N | tenant: null/tenantId；organizationNode/store: 對應 ID | IDX(建議) |
| effect | string/enum | N | allow/deny | — |
| expiresAt | datetime | N | 過期時間（臨時授權） | IDX(建議) |
| createdBy | string | Y | 建立者 employeeId | IDX(建議) |
| createdAt | datetime | Y | 建立時間 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, subjectType, subjectId) | N | 查詢主體權限 |
| (tenantId, roleId) | N | 角色被指派查詢 |
| (tenantId, scopeType, scopeId) | N | 按範圍過濾 |
| (tenantId, expiresAt) | N | 過期清理 |

## G. Data Policy（資料範圍）

### permission_policies

用途：對單一 Permission 附加資料限制規則。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, name) |
| name | string | Y | Policy 名稱 | IDX(tenantId, name) |
| description | string | N | 說明 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, name) | Y | Policy 名稱唯一（按需求） |

### policy_rules

用途：Policy 的規則明細。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, policyId) |
| policyId | string | Y | 關聯 policy | IDX(tenantId, policyId) |
| resource | string | Y | 例：orders | IDX(建議) |
| action | string | Y | 例：read | IDX(建議) |
| ruleType | string/enum | Y | ALL/SELF/STORE/ORGANIZATION_NODE/ORGANIZATION_SUBTREE/STORE_AND_SELF | IDX(建議) |
| conditions | json | N | 額外條件表達式 | — |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, policyId) | N | Policy 規則查詢 |
| (resource, action) | N | 規則過濾 |

### assignment_policies

用途：把 policy 掛到 role_assignment。

| 欄位 | 類型 | 必填 | 屬性/說明 | 索引 |
| --- | --- | --- | --- | --- |
| id | string | Y | 主鍵 | PK |
| tenantId | string | Y | 所屬租戶 | IDX(tenantId, roleAssignmentId) |
| roleAssignmentId | string | Y | 關聯 role_assignments | IDX(tenantId, roleAssignmentId) |
| policyId | string | Y | 關聯 permission_policies | IDX(建議) |

索引建議：

| 索引 | 唯一 | 說明 |
| --- | --- | --- |
| (tenantId, roleAssignmentId) | N | 尋找 assignment 的 policy |
| (tenantId, roleAssignmentId, policyId) | Y | 防止重複掛載 |

## H. 組織關係與權限計算（規則說明）

- 組織樹：`organization_nodes` 透過 `parentId`/`path`。
- 人員匯報樹：`employees.reportToEmployeeId`。
- 主管查下屬：可新增 `ruleType = REPORTING_TREE`（配合 reportToEmployeeId 展開）。

權限計算（簡要）：

1) 取得登入者有效 `role_assignments`（未過期）。
2) `assignment -> role -> role_permissions -> permissions` 展開權限。
3) 比對 `permission.key`，再按 `scopeType/scopeId` 判斷資料範圍。
4) 若有 deny 機制：deny 優先於 allow。
5) 可拆為：Guard（功能授權）與 Query Filter Builder（資料過濾）。

## I. Repository 設計藍圖

以下為常見用途與核心接口示意，目的是讓 Service 層有一致依賴。

### 1) Tenants Repository

常見用途：超管建立租戶、讀取租戶設定、登入時確認租戶狀態、系統初始化預設角色/權限。

```
Create(ctx, tenant) (tenantID, err)
GetByID(ctx, tenantID) (tenant, err)
GetBySlugOrCode(ctx, code) (tenant, err)
Update(ctx, tenantID, patch) err
SetStatus(ctx, tenantID, status) err
List(ctx, q) (ListResult[Tenant], err)
```

### 2) OrganizationNodes Repository（組織樹核心）

常見用途：顯示組織樹、拖拉移動節點（改 parentId + 重算 path）、查子樹（RBAC scope = orgSubtree）、設定部門主管。

```
Create(ctx, node) (orgNodeID, err)
GetByID(ctx, tenantID, orgNodeID) (node, err)
Update(ctx, tenantID, orgNodeID, patch) err
Delete(ctx, tenantID, orgNodeID) err
SetStatus(ctx, tenantID, orgNodeID, status) err

ListAll(ctx, tenantID, status) ([]OrgNode, err)
ListByParent(ctx, tenantID, parentID, status) ([]OrgNode, err)
ListSubtreeByPathPrefix(ctx, tenantID, pathPrefix, status) ([]OrgNode, err)
ListSubtreeByNodeID(ctx, tenantID, orgNodeID, status) ([]OrgNode, err)
GetAncestorsByPath(ctx, tenantID, orgNodeID) ([]OrgNode, err)

UpdateParentAndRepath(ctx, tenantID, orgNodeID, newParentID, newPath, newDepth) err
BulkUpdateRepathByPrefix(ctx, tenantID, oldPrefix, newPrefix, depthDelta) (modifiedCount, err)
UpdateSort(ctx, tenantID, orgNodeID, sort) err
IncChildrenCount(ctx, tenantID, orgNodeID, delta) err
```

### 3) Stores Repository（多店）

常見用途：店鋪管理、用 orgNodeId 查店、組織搬家時同步檢查店的 orgNodeId 是否仍有效。

```
Create(ctx, store) (storeID, err)
GetByID(ctx, tenantID, storeID) (store, err)
GetByCode(ctx, tenantID, code) (store, err)
Update(ctx, tenantID, storeID, patch) err
SetStatus(ctx, tenantID, storeID, status) err
List(ctx, tenantID, filter, page) (ListResult[Store], err)
ListByOrgNode(ctx, tenantID, orgNodeID, includeSubtree bool) ([]Store, err)
BatchGetByIDs(ctx, tenantID, storeIDs) ([]Store, err)
```

### 4) Employees Repository

常見用途：帳號管理、停權、重設密碼、查員工所屬部門/店、主管鏈、RBAC 計算需要基本資訊。

```
Create(ctx, emp) (employeeID, err)
GetByID(ctx, tenantID, employeeID) (emp, err)
GetByAccount(ctx, tenantID, account) (emp, err)
Update(ctx, tenantID, employeeID, patch) err
SetStatus(ctx, tenantID, employeeID, status) err
UpdatePasswordHash(ctx, tenantID, employeeID, passwordHash) err

SetPrimaryOrg(ctx, tenantID, employeeID, orgNodeID) err
SetPrimaryStore(ctx, tenantID, employeeID, storeID) err
SetReportTo(ctx, tenantID, employeeID, reportToEmployeeID) err

List(ctx, tenantID, filter, page) (ListResult[Employee], err)
ListByOrgNode(ctx, tenantID, orgNodeID, includeSubtree bool, page) (ListResult[Employee], err)
ListByStore(ctx, tenantID, storeID, page) (ListResult[Employee], err)
BatchGetByIDs(ctx, tenantID, employeeIDs) ([]Employee, err)
```

### 5) EmployeeOrgMembership Repository

常見用途：一個人掛多個部門、權限 scope=orgSubtree 可用它做可見節點/資料範圍補充。

```
Add(ctx, tenantID, employeeID, orgNodeID, roleInOrg) err
Remove(ctx, tenantID, employeeID, orgNodeID) err
ListByEmployee(ctx, tenantID, employeeID) ([]Membership, err)
ListByOrgNode(ctx, tenantID, orgNodeID) ([]Membership, err)
Exists(ctx, tenantID, employeeID, orgNodeID) (bool, err)
ReplaceAllForEmployee(ctx, tenantID, employeeID, orgNodeIDs) err
BatchListByEmployees(ctx, tenantID, employeeIDs) (map[EmployeeID][]Membership, err)
```

### 6) EmployeeStoreMembership Repository

常見用途：批貨員工支援多店、後台管理員負責多家店。

```
Add(ctx, tenantID, employeeID, storeID, position) err
Remove(ctx, tenantID, employeeID, storeID) err
ListByEmployee(ctx, tenantID, employeeID) ([]Membership, err)
ListByStore(ctx, tenantID, storeID) ([]Membership, err)
Exists(ctx, tenantID, employeeID, storeID) (bool, err)
ReplaceAllForEmployee(ctx, tenantID, employeeID, storeIDs) err
BatchListByEmployees(ctx, tenantID, employeeIDs) (map[EmployeeID][]Membership, err)
```

### 7) Permissions Repository

常見用途：系統內建 permission 初始化、後台查詢清單、RBAC 計算需要 permission key。

```
Create(ctx, perm) (permissionID, err)
GetByID(ctx, tenantIDOrNull, permissionID) (perm, err)
GetByKey(ctx, tenantIDOrNull, key) (perm, err)
List(ctx, tenantIDOrNull, filter, page) (ListResult[Permission], err)
BatchGetByIDs(ctx, tenantIDOrNull, permissionIDs) ([]Permission, err)
ListByKeys(ctx, tenantIDOrNull, keys) ([]Permission, err)
```

多租戶建議：`global perms tenantId=null` + `tenant custom perms tenantId=xxx`。

### 8) Roles Repository

常見用途：建角色、改名、停用、列出角色（用於指派）、RBAC 計算需要 role 資訊。

```
Create(ctx, role) (roleID, err)
GetByID(ctx, tenantID, roleID) (role, err)
Update(ctx, tenantID, roleID, patch) err
SetStatus(ctx, tenantID, roleID, status) err
Delete(ctx, tenantID, roleID) err
List(ctx, tenantID, filter, page) (ListResult[Role], err)
BatchGetByIDs(ctx, tenantID, roleIDs) ([]Role, err)
```

### 9) RolePermissions Repository（role -> permission 對照表）

常見用途：後台配置角色權限（add/remove/replace）、RBAC 計算時由 roleId 展開 permissionIds。

```
Add(ctx, tenantID, roleID, permissionID) err
Remove(ctx, tenantID, roleID, permissionID) err
ReplaceAll(ctx, tenantID, roleID, permissionIDs) err
ListPermissionIDsByRole(ctx, tenantID, roleID) ([]PermissionID, err)
ListByRoleIDs(ctx, tenantID, roleIDs) (map[RoleID][]PermissionID, err)
HasPermission(ctx, tenantID, roleID, permissionKeyOrID) (bool, err)
```

### 10) RoleAssignments Repository（人 -> 角色 + scope）

這是整個系統「有效權限」的核心資料來源。

```
Create(ctx, assignment) (assignmentID, err)
GetByID(ctx, tenantID, assignmentID) (assignment, err)
Update(ctx, tenantID, assignmentID, patch) err
Delete(ctx, tenantID, assignmentID) err
SetExpiresAt(ctx, tenantID, assignmentID, expiresAt) err

ListByEmployee(ctx, tenantID, employeeID) ([]RoleAssignment, err)
ListActiveByEmployee(ctx, tenantID, employeeID, atTime) ([]RoleAssignment, err)
ListByRole(ctx, tenantID, roleID, page) (ListResult[RoleAssignment], err)
ListByScope(ctx, tenantID, scopeType, scopeID, page) (ListResult[RoleAssignment], err)
ExistsDuplicate(ctx, tenantID, employeeID, roleID, scopeType, scopeID) (bool, err)

ListEmployeesByRoleAndScope(ctx, tenantID, roleID, scopeType, scopeID) ([]EmployeeID, err)
```

### 11) Effective RBAC Repository（可選）

若希望避免每次權限判斷都打多表，可做一個預計算集合：`employee_effective_access`。

欄位：`employeeId`, `tenantId`, `permissions: {key, scopes[]}`, `updatedAt`。

```
Upsert(ctx, tenantID, employeeID, effectiveAccess) err
Get(ctx, tenantID, employeeID) (effectiveAccess, err)
Delete(ctx, tenantID, employeeID) err
```

Service 於角色/指派/role_permissions 變更後重新計算並寫入（或只用 memory cache）。

## J. Service 組合與責任

建議至少包含以下 Service（因跨 repo）:

### OrgService

- GetTree(tenantID, visibleScope?)：`OrgNodeRepo.ListAll` + 組樹
- MoveNode(nodeID, newParentID, newSort)：`UpdateParent + BulkUpdateRepath`
- ListVisibleSubtree(employeeID)：RBAC scope 套 path 查詢

### RBACService

- GetEmployeeAssignments(employeeID)
- ComputeEffectivePermissions(employeeID)
- HasPermission(employeeID, permissionKey)
- BuildDataFilter(employeeID, permissionKey)

### EmployeeService

- CreateEmployee + memberships + assignments（一次建立完整員工）
- UpdateEmployeeProfile
- SetEmployeeOrgs / SetEmployeeStores

### RoleService

- CreateRole
- UpdateRolePermissions (ReplaceAll)
- AssignRoleToEmployee (Create assignment)
- RevokeRoleAssignment

## K. Controller / API 範例

建議的後台 API 對應：

- OrgController
  - `GET /org-nodes/tree`
  - `GET /org-nodes?parentId=...`
  - `POST /org-nodes`
  - `PATCH /org-nodes/:id`
  - `POST /org-nodes/:id/move`
- StoreController
- EmployeeController
- RoleController
- PermissionController
- AssignmentController（指派角色到人、查人有哪些角色）

## L. 內容改善建議（持續維護）

- 新增「權限實例」：列出 2-3 個完整案例（角色、權限、範圍、查詢結果）。
- 新增「資料遷移策略」：包含欄位演進、索引調整與回填方式。
- 補上「稽核/審計」：如 `createdBy`, `updatedBy`, `audit_log`。
- 統一「枚舉字典」：收斂各 table 的 enum 值與對應定義。
- 針對查詢慢點建立「查詢指標」與「慢查治理」清單。

## 資料結構關係圖

```mermaid
erDiagram
  TENANTS {
    string id PK
    string name
    string status
    datetime createdAt
    datetime updatedAt
  }

  ORGANIZATION_NODES {
    string id PK
    string tenantId
    string name
    string type
    string parentId
    string path
    int depth
    string managerEmployeeId
    string status
    datetime createdAt
    datetime updatedAt
  }

  STORES {
    string id PK
    string tenantId
    string code
    string name
    string organizationNodeId
    string region
    string status
    string parentStoreId
    string path
    datetime createdAt
    datetime updatedAt
  }

  EMPLOYEES {
    string id PK
    string tenantId
    string account
    string passwordHash
    string displayName
    string status
    string primaryOrganizationNodeId
    string primaryStoreId
    string jobTitle
    string reportToEmployeeId
    datetime createdAt
    datetime updatedAt
  }

  EMPLOYEE_ORG_MEMBERSHIPS {
    string id PK
    string tenantId
    string employeeId
    string organizationNodeId
    string roleInOrganization
    datetime createdAt
  }

  EMPLOYEE_STORE_MEMBERSHIPS {
    string id PK
    string tenantId
    string employeeId
    string storeId
    string position
    datetime createdAt
  }

  PERMISSIONS {
    string id PK
    string tenantId
    string key
    string resource
    string action
    string description
    string status
  }

  ROLES {
    string id PK
    string tenantId
    string name
    string code
    bool isSystem
    string status
    datetime createdAt
    datetime updatedAt
  }

  ROLE_PERMISSIONS {
    string id PK
    string tenantId
    string roleId
    string permissionId
    datetime createdAt
  }

  ROLE_ASSIGNMENTS {
    string id PK
    string tenantId
    string subjectType
    string subjectId
    string roleId
    string scopeType
    string scopeId
    string effect
    datetime expiresAt
    string createdBy
    datetime createdAt
  }

  PERMISSION_POLICIES {
    string id PK
    string tenantId
    string name
    string description
  }

  POLICY_RULES {
    string id PK
    string tenantId
    string policyId
    string resource
    string action
    string ruleType
    json conditions
  }

  ASSIGNMENT_POLICIES {
    string id PK
    string tenantId
    string roleAssignmentId
    string policyId
  }

  TENANTS ||--o{ ORGANIZATION_NODES : has
  TENANTS ||--o{ STORES : has
  TENANTS ||--o{ EMPLOYEES : has
  TENANTS ||--o{ ROLES : has
  TENANTS ||--o{ PERMISSIONS : "custom (tenantId nullable)"
  TENANTS ||--o{ ROLE_ASSIGNMENTS : assigns
  TENANTS ||--o{ PERMISSION_POLICIES : has
  TENANTS ||--o{ POLICY_RULES : has
  TENANTS ||--o{ ASSIGNMENT_POLICIES : has
  TENANTS ||--o{ ROLE_PERMISSIONS : has
  TENANTS ||--o{ EMPLOYEE_ORG_MEMBERSHIPS : has
  TENANTS ||--o{ EMPLOYEE_STORE_MEMBERSHIPS : has

  ORGANIZATION_NODES ||--o{ ORGANIZATION_NODES : parent
  ORGANIZATION_NODES ||--o{ STORES : owns
  ORGANIZATION_NODES ||--o{ EMPLOYEES : primary
  ORGANIZATION_NODES ||--o{ EMPLOYEE_ORG_MEMBERSHIPS : has
  EMPLOYEES ||--o{ ORGANIZATION_NODES : manages

  STORES ||--o{ STORES : parent
  STORES ||--o{ EMPLOYEE_STORE_MEMBERSHIPS : has
  STORES ||--o{ EMPLOYEES : "primary optional"

  EMPLOYEES ||--o{ EMPLOYEE_ORG_MEMBERSHIPS : member
  EMPLOYEES ||--o{ EMPLOYEE_STORE_MEMBERSHIPS : works_at
  EMPLOYEES ||--o{ EMPLOYEES : reports_to
  EMPLOYEES ||--o{ ROLE_ASSIGNMENTS : subject(employee)

  ROLES ||--o{ ROLE_PERMISSIONS : grants
  PERMISSIONS ||--o{ ROLE_PERMISSIONS : uses
  ROLES ||--o{ ROLE_ASSIGNMENTS : assigned

  ROLE_ASSIGNMENTS ||--o{ ASSIGNMENT_POLICIES : has
  PERMISSION_POLICIES ||--o{ POLICY_RULES : contains
  PERMISSION_POLICIES ||--o{ ASSIGNMENT_POLICIES : applied

  ROLE_ASSIGNMENTS }o--|| TENANTS : scope(tenant)
  ROLE_ASSIGNMENTS }o--|| ORGANIZATION_NODES : scope(org)
  ROLE_ASSIGNMENTS }o--|| STORES : scope(store)
```
