package permission

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	corev1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/core/v1alpha"
	registrationv1alpha "github.com/EdgeNet-project/edgenet/pkg/apis/registration/v1alpha"
	"github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenettestclient "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/fake"
	"github.com/EdgeNet-project/edgenet/pkg/util"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type TestGroup struct {
	tenant        corev1alpha.Tenant
	user          registrationv1alpha.UserRequest
	client        kubernetes.Interface
	edgenetclient versioned.Interface
	namespace     corev1.Namespace
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func (g *TestGroup) Init() {
	tenantObj := corev1alpha.Tenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Tenant",
			APIVersion: "core.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "edgenet",
		},
		Spec: corev1alpha.TenantSpec{
			FullName:  "EdgeNet",
			ShortName: "EdgeNet",
			URL:       "https://www.edge-net.org",
			Address: corev1alpha.Address{
				City:    "Paris - NY - CA",
				Country: "France - US",
				Street:  "4 place Jussieu, boite 169",
				ZIP:     "75005",
			},
			Contact: corev1alpha.Contact{
				Email:     "john.doe@edge-net.org",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+333333333",
				Username:  "johndoe",
			},
			Enabled: false,
		},
	}
	userObj := registrationv1alpha.UserRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserRequest",
			APIVersion: "registration.edgenet.io/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "johnsmith",
		},
		Spec: registrationv1alpha.UserRequestSpec{
			Tenant:    "edgenet",
			FirstName: "John",
			LastName:  "Smith",
			Email:     "john.smith@edge-net.org",
			Role:      "Collaborator",
		},
	}
	g.tenant = tenantObj
	g.user = userObj
	g.client = testclient.NewSimpleClientset()
	g.edgenetclient = edgenettestclient.NewSimpleClientset()
	Clientset = g.client
	g.namespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s", g.tenant.GetName())}}
	g.client.CoreV1().Namespaces().Create(context.TODO(), &g.namespace, metav1.CreateOptions{})
}

func TestCreateClusterRoles(t *testing.T) {
	g := TestGroup{}
	g.Init()

	err := CreateClusterRoles()
	util.OK(t, err)

	cases := map[string]struct {
		expected string
	}{
		"default tenant owner": {"edgenet:tenant-owner"},
		"default tenant admin": {"edgenet:tenant-admin"},
		"default collaborator": {"edgenet:tenant-collaborator"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			_, err = g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}

	tenant := g.tenant
	ownerUser := registrationv1alpha.UserRequest{}
	ownerUser.SetName(strings.ToLower(tenant.Spec.Contact.Username))
	ownerUser.Spec.Email = tenant.Spec.Contact.Email
	ownerUser.Spec.FirstName = tenant.Spec.Contact.FirstName
	ownerUser.Spec.LastName = tenant.Spec.Contact.LastName
	ownerUser.Spec.Role = "Owner"
	ownerUser.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})
	user1 := g.user
	user1.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})
	user2 := g.user
	user2.SetName("joepublic")
	user2.Spec.FirstName = "Joe"
	user2.Spec.LastName = "Public"
	user2.Spec.Email = "joe.public@edge-net.org"
	user2.Spec.Role = "Admin"
	user2.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})

	t.Run("role binding", func(t *testing.T) {
		cases := map[string]struct {
			tenant    string
			namespace string
			roleName  string
			user      registrationv1alpha.UserRequest
			expected  string
		}{
			"owner":        {tenant.GetName(), tenant.GetName(), "edgenet:tenant-owner", ownerUser, fmt.Sprintf("edgenet:tenant-owner-%s", ownerUser.GetName())},
			"collaborator": {tenant.GetName(), tenant.GetName(), "edgenet:tenant-collaborator", user1, fmt.Sprintf("edgenet:tenant-collaborator-%s", user1.GetName())},
			"admin":        {tenant.GetName(), tenant.GetName(), "edgenet:tenant-admin", user2, fmt.Sprintf("edgenet:tenant-admin-%s", user2.GetName())},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				userLabels := tc.user.GetLabels()
				CreateObjectSpecificRoleBinding(tc.tenant, tc.namespace, tc.roleName, tc.user.DeepCopy())
				_, err := g.client.RbacV1().RoleBindings(tenant.GetName()).Get(context.TODO(), fmt.Sprintf("%s-%s", tc.expected, userLabels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
				util.OK(t, err)
				err = CreateObjectSpecificRoleBinding(tc.tenant, tc.namespace, tc.roleName, tc.user.DeepCopy())
				util.OK(t, err)
			})
		}
	})
	err = CreateClusterRoles()
	util.OK(t, err)
}

func TestCreateObjectSpecificClusterRole(t *testing.T) {
	g := TestGroup{}
	g.Init()

	tenant1 := g.tenant
	ownerUser := registrationv1alpha.UserRequest{}
	ownerUser.SetName(strings.ToLower(tenant1.Spec.Contact.Username))
	ownerUser.Spec.Email = tenant1.Spec.Contact.Email
	ownerUser.Spec.FirstName = tenant1.Spec.Contact.FirstName
	ownerUser.Spec.LastName = tenant1.Spec.Contact.LastName
	ownerUser.Spec.Role = "Owner"
	ownerUser.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})
	tenant2 := g.tenant
	tenant2.SetName("lip6")
	user := registrationv1alpha.UserRequest{}
	user.SetName(g.user.GetName())
	user.Spec.Email = g.user.Spec.Email
	user.Spec.FirstName = g.user.Spec.FirstName
	user.Spec.LastName = g.user.Spec.LastName
	user.Spec.Role = g.user.Spec.Role
	user.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})

	cases := map[string]struct {
		tenant       corev1alpha.Tenant
		apiGroup     string
		resource     string
		resourceName string
		verbs        []string
		expected     string
	}{
		"tenant":                    {tenant1, "core.edgenet.io", "tenants", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:%s:tenants:%s-name", tenant1.GetName(), tenant1.GetName())},
		"tenant resource quota":     {tenant1, "core.edgenet.io", "tenantresourcequotas", tenant1.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:%s:tenantresourcequotas:%s-name", tenant1.GetName(), tenant1.GetName())},
		"node contribution":         {tenant2, "core.edgenet.io", "nodecontributions", "ple", []string{"get", "update", "patch", "delete"}, fmt.Sprintf("edgenet:%s:nodecontributions:%s-name", tenant2.GetName(), "ple")},
		"user registration request": {tenant1, "registration.edgenet.io", "userrequests", user.GetName(), []string{"get", "update", "patch", "delete"}, fmt.Sprintf("edgenet:%s:userrequests:%s-name", tenant1.GetName(), user.GetName())},
		"email verification":        {tenant2, "registration.edgenet.io", "emailverifications", "abcdefghi", []string{"get", "update", "patch", "delete"}, fmt.Sprintf("edgenet:%s:emailverifications:%s-name", tenant2.GetName(), "abcdefghi")},
		"acceptable use policy":     {tenant1, "core.edgenet.io", "acceptableusepolicies", ownerUser.GetName(), []string{"get", "update", "patch"}, fmt.Sprintf("edgenet:%s:acceptableusepolicies:%s-name", tenant1.GetName(), ownerUser.GetName())},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			CreateObjectSpecificClusterRole(tc.tenant.GetName(), tc.apiGroup, tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			clusterRole, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
			if err == nil {
				util.Equals(t, tc.verbs, clusterRole.Rules[0].Verbs)
			}
			err = CreateObjectSpecificClusterRole(tc.tenant.GetName(), tc.apiGroup, tc.resource, tc.resourceName, "name", tc.verbs, []metav1.OwnerReference{})
			util.OK(t, err)
		})
	}

	t.Run("cluster role binding", func(t *testing.T) {
		cases := map[string]struct {
			tenant   string
			roleName string
			user     registrationv1alpha.UserRequest
			expected string
		}{
			"tenant":                    {tenant1.GetName(), fmt.Sprintf("%s-tenants-%s", tenant1.GetName(), tenant1.GetName()), ownerUser, fmt.Sprintf("%s-tenants-%s-%s", tenant1.GetName(), tenant1.GetName(), ownerUser.GetName())},
			"tenant resource quota":     {tenant1.GetName(), fmt.Sprintf("%s-tenantresourcequotas-%s", tenant1.GetName(), tenant1.GetName()), ownerUser, fmt.Sprintf("%s-tenantresourcequotas-%s-%s", tenant1.GetName(), tenant1.GetName(), ownerUser.GetName())},
			"node contribution":         {tenant1.GetName(), fmt.Sprintf("%s-nodecontributions-%s", tenant1.GetName(), "ple"), ownerUser, fmt.Sprintf("%s-nodecontributions-%s-%s", tenant1.GetName(), "ple", ownerUser.GetName())},
			"user registration request": {tenant1.GetName(), fmt.Sprintf("%s-userrequests-%s", tenant1.GetName(), user.GetName()), ownerUser, fmt.Sprintf("%s-userrequests-%s-%s", tenant1.GetName(), user.GetName(), ownerUser.GetName())},
			"email verification":        {tenant1.GetName(), fmt.Sprintf("%s-emailverifications-%s", tenant1.GetName(), "abcdefghi"), user, fmt.Sprintf("%s-emailverifications-%s-%s", tenant1.GetName(), "abcdefghi", user.GetName())},
			"acceptable use policy":     {tenant1.GetName(), fmt.Sprintf("%s-acceptableusepolicies-%s", tenant1.GetName(), ownerUser.GetName()), ownerUser, fmt.Sprintf("%s-acceptableusepolicies-%s-%s", tenant1.GetName(), ownerUser.GetName(), ownerUser.GetName())},
		}
		for k, tc := range cases {
			t.Run(k, func(t *testing.T) {
				userLabels := tc.user.GetLabels()
				roleBindLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": tc.tenant, "edge-net.io/identity": "true", "edge-net.io/username": tc.user.GetName(),
					"edge-net.io/user-template-hash": userLabels["edge-net.io/user-template-hash"], "edge-net.io/firstname": tc.user.Spec.FirstName, "edge-net.io/lastname": tc.user.Spec.LastName, "edge-net.io/role": tc.user.Spec.Role}
				CreateObjectSpecificClusterRoleBinding(tc.tenant, tc.roleName, fmt.Sprintf("%s-%s", tc.user.GetName(), userLabels["edge-net.io/user-template-hash"]), tc.user.Spec.Email, roleBindLabels, []metav1.OwnerReference{})
				_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("%s-%s", tc.expected, userLabels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
				util.OK(t, err)
				err = CreateObjectSpecificClusterRoleBinding(tc.tenant, tc.roleName, fmt.Sprintf("%s-%s", tc.user.GetName(), userLabels["edge-net.io/user-template-hash"]), tc.user.Spec.Email, roleBindLabels, []metav1.OwnerReference{})
				util.OK(t, err)
			})
		}
	})
}

func TestPermissionSystem(t *testing.T) {
	g := TestGroup{}
	g.Init()

	tenant := g.tenant
	user1 := registrationv1alpha.UserRequest{}
	user1.SetName(strings.ToLower(tenant.Spec.Contact.Username))
	user1.Spec.Email = tenant.Spec.Contact.Email
	user1.Spec.FirstName = tenant.Spec.Contact.FirstName
	user1.Spec.LastName = tenant.Spec.Contact.LastName
	user1.Spec.Role = "Owner"
	user1.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})
	user2 := g.user
	user3 := g.user
	user3.SetName("joepublic")
	user3.Spec.FirstName = "Joe"
	user3.Spec.LastName = "Public"
	user3.Spec.Email = "joe.public@edge-net.org"
	user3.Spec.Role = "Admin"
	user3.SetLabels(map[string]string{"edge-net.io/user-template-hash": util.GenerateRandomString(6)})

	err := CreateClusterRoles()
	util.OK(t, err)
	cases := map[string]struct {
		expected string
	}{
		"create cluster role for tenant owner":         {"edgenet:tenant-owner"},
		"create cluster role for default collaborator": {"edgenet:tenant-collaborator"},
		"create cluster role for default tenant admin": {"edgenet:tenant-admin"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), tc.expected, metav1.GetOptions{})
			util.OK(t, err)
		})
	}
	t.Run("bind cluster role for tenant owner", func(t *testing.T) {
		CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "edgenet:tenant-owner", user1.DeepCopy())
	})
	t.Run("bind cluster role for tenant collaborator", func(t *testing.T) {
		CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "edgenet:tenant-collaborator", user2.DeepCopy())
	})
	t.Run("bind cluster role for tenant admin", func(t *testing.T) {
		CreateObjectSpecificRoleBinding(tenant.GetName(), tenant.GetName(), "edgenet:tenant-admin", user3.DeepCopy())
	})

	t.Run("create owner specific tenant role", func(t *testing.T) {
		CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "owner", []string{"get", "update", "patch"}, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create admin specific tenant role", func(t *testing.T) {
		CreateObjectSpecificClusterRole(tenant.GetName(), "core.edgenet.io", "tenants", tenant.GetName(), "admin", []string{"get"}, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoles().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-admin", tenant.GetName(), tenant.GetName()), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create owner role binding", func(t *testing.T) {
		userLabels := user1.GetLabels()
		roleBindLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/identity": "true", "edge-net.io/username": user1.GetName(),
			"edge-net.io/user-template-hash": userLabels["edge-net.io/user-template-hash"], "edge-net.io/firstname": user1.Spec.FirstName, "edge-net.io/lastname": user1.Spec.LastName, "edge-net.io/role": user1.Spec.Role}

		CreateObjectSpecificClusterRoleBinding(tenant.GetName(), fmt.Sprintf("edgenet:%s:tenants:%s-owner", tenant.GetName(), tenant.GetName()), fmt.Sprintf("%s-%s", user1.GetName(), userLabels["edge-net.io/user-template-hash"]), user1.Spec.Email, roleBindLabels, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-owner-%s-%s", tenant.GetName(), tenant.GetName(), user1.GetName(), userLabels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
		util.OK(t, err)
	})
	t.Run("create admin role binding", func(t *testing.T) {
		userLabels := user1.GetLabels()
		roleBindLabels := map[string]string{"edge-net.io/generated": "true", "edge-net.io/tenant": tenant.GetName(), "edge-net.io/identity": "true", "edge-net.io/username": user3.GetName(),
			"edge-net.io/user-template-hash": userLabels["edge-net.io/user-template-hash"], "edge-net.io/firstname": user3.Spec.FirstName, "edge-net.io/lastname": user3.Spec.LastName, "edge-net.io/role": user3.Spec.Role}

		CreateObjectSpecificClusterRoleBinding(tenant.GetName(), fmt.Sprintf("edgenet:%s:tenants:%s-admin", tenant.GetName(), tenant.GetName()), fmt.Sprintf("%s-%s", user3.GetName(), userLabels["edge-net.io/user-template-hash"]), user3.Spec.Email, roleBindLabels, []metav1.OwnerReference{})
		_, err := g.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), fmt.Sprintf("edgenet:%s:tenants:%s-admin-%s-%s", tenant.GetName(), tenant.GetName(), user3.GetName(), userLabels["edge-net.io/user-template-hash"]), metav1.GetOptions{})
		util.OK(t, err)
	})

	permissionCases := map[string]struct {
		user         registrationv1alpha.UserRequest
		namespace    string
		resource     string
		resourceName string
		scope        string
		expected     bool
	}{
		"owner/authorized for subnamespace":          {user1, g.namespace.GetName(), "subnamespaces", "", "namespace", true},
		"collaborator/authorized for subnamespace":   {user2, g.namespace.GetName(), "subnamespaces", "", "namespace", false},
		"owner/authorized for roles":                 {user1, g.namespace.GetName(), "roles", "", "namespace", true},
		"collaborator/authorized for roles":          {user2, g.namespace.GetName(), "roles", "", "namespace", false},
		"owner/authorized for role bindings":         {user1, g.namespace.GetName(), "rolebindings", "", "namespace", true},
		"admin/authorized for role bindings":         {user3, g.namespace.GetName(), "rolebindings", "", "namespace", true},
		"owner/authorized for cluster role bindings": {user1, "", "clusterrolebindings", "", "cluster", false},
		"owner/authorized for tenant object":         {user1, "", "tenants", tenant.GetName(), "cluster", true},
		"collaborator/authorized for tenant object":  {user2, "", "tenants", tenant.GetName(), "cluster", false},
		"admin/authorized for tenant object":         {user3, "", "tenants", tenant.GetName(), "cluster", false},
	}
	for k, tc := range permissionCases {
		t.Run(k, func(t *testing.T) {
			authorized := CheckAuthorization(tc.namespace, tc.user.Spec.Email, tc.resource, tc.resourceName, tc.scope)
			util.Equals(t, tc.expected, authorized)
		})
	}
}
