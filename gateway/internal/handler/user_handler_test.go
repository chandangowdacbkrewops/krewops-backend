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
	listWorkTypes func(context.Context, *userv1.ListWorkTypesRequest, ...grpc.CallOption) (*userv1.ListWorkTypesResponse, error)
}

func (s userServiceClientStub) GetProfile(context.Context, *userv1.GetProfileRequest, ...grpc.CallOption) (*userv1.GetProfileResponse, error) {
	return nil, errors.New("not implemented")
}

func (s userServiceClientStub) CreateProfile(context.Context, *userv1.CreateProfileRequest, ...grpc.CallOption) (*userv1.CreateProfileResponse, error) {
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
