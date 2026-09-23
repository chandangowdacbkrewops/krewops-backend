package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/middleware"
	userv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/user/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type userServiceClientStub struct {
	listWorkTypes           func(context.Context, *userv1.ListWorkTypesRequest, ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error)
	listWorkCategories      func(context.Context, *userv1.ListWorkCategoriesRequest, ...grpc.CallOption) (*userv1.ListWorkCategoriesResponse, error)
	listWorkTypesByCategory func(context.Context, *userv1.ListWorkTypesByCategoryRequest, ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error)
	listWorkTypeFields      func(context.Context, *userv1.ListWorkTypeFieldsRequest, ...grpc.CallOption) (*userv1.ListWorkTypeFieldsResponse, error)
	createProfile           func(context.Context, *userv1.CreateProfileRequest, ...grpc.CallOption) (*userv1.CreateProfileResponse, error)
	updateProfile           func(context.Context, *userv1.UpdateProfileRequest, ...grpc.CallOption) (*userv1.UpdateProfileResponse, error)
}

func (s userServiceClientStub) GetProfile(context.Context, *userv1.GetProfileRequest, ...grpc.CallOption) (*userv1.GetProfileResponse, error) {
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) CreateProfile(ctx context.Context, req *userv1.CreateProfileRequest, opts ...grpc.CallOption) (*userv1.CreateProfileResponse, error) {
	if s.createProfile != nil {
		return s.createProfile(ctx, req, opts...)
	}
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest, opts ...grpc.CallOption) (*userv1.UpdateProfileResponse, error) {
	if s.updateProfile != nil {
		return s.updateProfile(ctx, req, opts...)
	}
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) CreateWorkerProfile(context.Context, *userv1.CreateWorkerProfileRequest, ...grpc.CallOption) (*userv1.CreateWorkerProfileResponse, error) {
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) CreateOwnerProfile(context.Context, *userv1.CreateOwnerProfileRequest, ...grpc.CallOption) (*userv1.CreateOwnerProfileResponse, error) {
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) ListBusinessTypes(context.Context, *userv1.ListBusinessTypesRequest, ...grpc.CallOption) (*userv1.ListBusinessTypesResponse, error) {
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) ListWorkTypes(ctx context.Context, req *userv1.ListWorkTypesRequest, opts ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error) {
	return s.listWorkTypes(ctx, req, opts...)
}

func (s userServiceClientStub) ListWorkCategories(ctx context.Context, req *userv1.ListWorkCategoriesRequest, opts ...grpc.CallOption) (*userv1.ListWorkCategoriesResponse, error) {
	if s.listWorkCategories != nil {
		return s.listWorkCategories(ctx, req, opts...)
	}
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) ListWorkTypesByCategory(ctx context.Context, req *userv1.ListWorkTypesByCategoryRequest, opts ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error) {
	if s.listWorkTypesByCategory != nil {
		return s.listWorkTypesByCategory(ctx, req, opts...)
	}
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) ListWorkTypeFields(ctx context.Context, req *userv1.ListWorkTypeFieldsRequest, opts ...grpc.CallOption) (*userv1.ListWorkTypeFieldsResponse, error) {
	if s.listWorkTypeFields != nil {
		return s.listWorkTypeFields(ctx, req, opts...)
	}
	return nil, errors.New("not implemented")
}

func TestUserHandlerListWorkTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkTypes: func(_ context.Context, req *userv1.ListWorkTypesRequest, _ ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error) {
			if req == nil {
				t.Fatal("ListWorkTypes request must not be nil")
			}
			return &userv1.ListWorkTypesResponse{WorkTypes: []*userv1.WorkType{{Id: "work-type-1", Name: "Carpenter"}}}, nil
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-types", nil)
	handler.ListWorkTypes(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	const want = `{"success":true,"data":{"work_types":[{"id":"work-type-1","name":"Carpenter"}]}`
	if len(recorder.Body.String()) < len(want) || recorder.Body.String()[:len(want)] != want {
		t.Fatalf("response = %s, want prefix %s", recorder.Body.String(), want)
	}
}

func TestUserHandlerListWorkTypesReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkTypes: func(context.Context, *userv1.ListWorkTypesRequest, ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error) {
			return nil, errors.New("user service unavailable")
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-types", nil)
	handler.ListWorkTypes(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	const want = `"message":"failed to fetch work types"`
	if response := recorder.Body.String(); !strings.Contains(response, want) {
		t.Fatalf("response = %s, want %s", response, want)
	}
}

func TestUserHandlerListWorkCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkCategories: func(_ context.Context, req *userv1.ListWorkCategoriesRequest, _ ...grpc.CallOption) (*userv1.ListWorkCategoriesResponse, error) {
			if req == nil {
				t.Fatal("ListWorkCategories request must not be nil")
			}
			return &userv1.ListWorkCategoriesResponse{Categories: []*userv1.WorkCategory{{Id: "category-1", Code: "construction", Name: "Construction"}}}, nil
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-categories", nil)
	handler.ListWorkCategories(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	const want = `{"success":true,"data":{"categories":[{"id":"category-1","code":"construction","name":"Construction"}]}`
	if len(recorder.Body.String()) < len(want) || recorder.Body.String()[:len(want)] != want {
		t.Fatalf("response = %s, want prefix %s", recorder.Body.String(), want)
	}
}

func TestUserHandlerListWorkCategoriesReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkCategories: func(context.Context, *userv1.ListWorkCategoriesRequest, ...grpc.CallOption) (*userv1.ListWorkCategoriesResponse, error) {
			return nil, errors.New("user service unavailable")
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-categories", nil)
	handler.ListWorkCategories(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	const want = `"message":"failed to fetch work categories"`
	if response := recorder.Body.String(); !strings.Contains(response, want) {
		t.Fatalf("response = %s, want %s", response, want)
	}
}

func TestUserHandlerListWorkTypesByCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkTypesByCategory: func(_ context.Context, req *userv1.ListWorkTypesByCategoryRequest, _ ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error) {
			if req.CategoryId != "category-1" {
				t.Fatalf("category_id = %q, want %q", req.CategoryId, "category-1")
			}
			return &userv1.ListWorkTypesResponse{WorkTypes: []*userv1.WorkType{{Id: "work-type-1", Name: "Carpenter", CategoryId: "category-1"}}}, nil
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-categories/category-1/work-types", nil)
	context.Params = gin.Params{{Key: "categoryId", Value: "category-1"}}
	handler.ListWorkTypesByCategory(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	const want = `{"success":true,"data":{"work_types":[{"id":"work-type-1","name":"Carpenter","category_id":"category-1"}]}`
	if len(recorder.Body.String()) < len(want) || recorder.Body.String()[:len(want)] != want {
		t.Fatalf("response = %s, want prefix %s", recorder.Body.String(), want)
	}
}

func TestUserHandlerListWorkTypesByCategoryReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkTypesByCategory: func(context.Context, *userv1.ListWorkTypesByCategoryRequest, ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error) {
			return nil, errors.New("user service unavailable")
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-categories/category-1/work-types", nil)
	context.Params = gin.Params{{Key: "categoryId", Value: "category-1"}}
	handler.ListWorkTypesByCategory(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	const want = `"message":"failed to fetch work types"`
	if response := recorder.Body.String(); !strings.Contains(response, want) {
		t.Fatalf("response = %s, want %s", response, want)
	}
}

func TestUserHandlerListWorkTypeFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkTypeFields: func(_ context.Context, req *userv1.ListWorkTypeFieldsRequest, _ ...grpc.CallOption) (*userv1.ListWorkTypeFieldsResponse, error) {
			if req.WorkTypeId != "work-type-1" {
				t.Fatalf("work_type_id = %q, want %q", req.WorkTypeId, "work-type-1")
			}
			return &userv1.ListWorkTypeFieldsResponse{Fields: []*userv1.WorkTypeField{{Id: "field-1", WorkTypeId: "work-type-1", FieldKey: "wire_gauge", Label: "Wire Gauge", FieldType: "TEXT"}}}, nil
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-types/work-type-1/fields", nil)
	context.Params = gin.Params{{Key: "workTypeId", Value: "work-type-1"}}
	handler.ListWorkTypeFields(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	const want = `{"success":true,"data":{"fields":[{"id":"field-1","work_type_id":"work-type-1","field_key":"wire_gauge","label":"Wire Gauge","field_type":"TEXT"}]}`
	if len(recorder.Body.String()) < len(want) || recorder.Body.String()[:len(want)] != want {
		t.Fatalf("response = %s, want prefix %s", recorder.Body.String(), want)
	}
}

func TestUserHandlerListWorkTypeFieldsReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		listWorkTypeFields: func(context.Context, *userv1.ListWorkTypeFieldsRequest, ...grpc.CallOption) (*userv1.ListWorkTypeFieldsResponse, error) {
			return nil, errors.New("user service unavailable")
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/work-types/work-type-1/fields", nil)
	context.Params = gin.Params{{Key: "workTypeId", Value: "work-type-1"}}
	handler.ListWorkTypeFields(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	const want = `"message":"failed to fetch work type fields"`
	if response := recorder.Body.String(); !strings.Contains(response, want) {
		t.Fatalf("response = %s, want %s", response, want)
	}
}

func TestUserHandlerUpdateProfileUsesAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		updateProfile: func(_ context.Context, req *userv1.UpdateProfileRequest, _ ...grpc.CallOption) (*userv1.UpdateProfileResponse, error) {
			if req.UserId != "user-123" {
				t.Fatalf("user_id = %q, want %q", req.UserId, "user-123")
			}
			if req.FirstName != "Ada" || req.LastName != "Lovelace" || req.Country != "India" || req.State != "Karnataka" || req.City != "Bengaluru" || req.PostalCode != "560001" {
				t.Fatalf("unexpected update request: %#v", req)
			}
			return &userv1.UpdateProfileResponse{Profile: &userv1.UserProfile{Id: "profile-123", FirstName: "Ada"}}, nil
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/v1/users/profile", strings.NewReader(`{"first_name":"Ada","last_name":"Lovelace","country":"India","state":"Karnataka","city":"Bengaluru","postal_code":"560001","user_type":"worker"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("user_id", "user-123")
	handler.UpdateProfile(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; response = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"first_name":"Ada"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestUserHandlerCreateProfileForwardsUserType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUserHandler(userServiceClientStub{
		createProfile: func(_ context.Context, req *userv1.CreateProfileRequest, _ ...grpc.CallOption) (*userv1.CreateProfileResponse, error) {
			if req.UserId != "user-123" {
				t.Fatalf("user_id = %q, want %q", req.UserId, "user-123")
			}
			if req.UserType != "worker" {
				t.Fatalf("user_type = %q, want %q", req.UserType, "worker")
			}
			return &userv1.CreateProfileResponse{Profile: &userv1.UserProfile{UserType: req.UserType}}, nil
		},
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users/profile", strings.NewReader(`{"first_name":"Ada","last_name":"Lovelace","country":"India","state":"Karnataka","city":"Bengaluru","postal_code":"560001","user_type":"worker"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("user_id", "user-123")
	handler.CreateProfile(context)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; response = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"user_type":"worker"`) {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}

func TestWorkTypesRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/api/v1/work-types",
		middleware.AuthMiddleware("test-secret"),
		NewUserHandler(userServiceClientStub{}).ListWorkTypes,
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/work-types", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestWorkCategoriesRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/api/v1/work-categories",
		middleware.AuthMiddleware("test-secret"),
		NewUserHandler(userServiceClientStub{}).ListWorkCategories,
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/work-categories", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestWorkTypesByCategoryRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/api/v1/work-categories/:categoryId/work-types",
		middleware.AuthMiddleware("test-secret"),
		NewUserHandler(userServiceClientStub{}).ListWorkTypesByCategory,
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/work-categories/category-1/work-types", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestWorkTypeFieldsRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/api/v1/work-types/:workTypeId/fields",
		middleware.AuthMiddleware("test-secret"),
		NewUserHandler(userServiceClientStub{}).ListWorkTypeFields,
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/work-types/work-type-1/fields", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
