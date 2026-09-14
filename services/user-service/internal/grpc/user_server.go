package grpcserver

import (
	"context"
	"errors"

	userv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/user/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	userv1.UnimplementedUserServiceServer

	userService *service.UserService
}

func NewUserServer(
	userService *service.UserService,
) *UserServer {

	return &UserServer{
		userService: userService,
	}
}

func (s *UserServer) GetProfile(
	ctx context.Context,
	req *userv1.GetProfileRequest,
) (*userv1.GetProfileResponse, error) {

	profile, err :=
		s.userService.GetProfile(
			ctx,
			req.UserId,
		)

	if err != nil {
		return nil, err
	}

	return &userv1.GetProfileResponse{
		Profile: toUserProfile(profile),
	}, nil
}

func (s *UserServer) CreateProfile(
	ctx context.Context,
	req *userv1.CreateProfileRequest,
) (*userv1.CreateProfileResponse, error) {

	profile, err :=
		s.userService.CreateProfile(
			ctx,
			req.UserId,
			model.CreateProfileRequest{
				FirstName:  req.FirstName,
				LastName:   req.LastName,
				Country:    req.Country,
				State:      req.State,
				City:       req.City,
				PostalCode: req.PostalCode,
			},
		)

	if err != nil {
		if errors.Is(err, repository.ErrProfileAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "profile already exists")
		}
		if errors.Is(err, service.ErrInvalidProfile) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "unable to create profile")
	}

	return &userv1.CreateProfileResponse{
		Profile: toUserProfile(profile),
	}, nil
}

func (s *UserServer) CreateWorkerProfile(
	ctx context.Context,
	req *userv1.CreateWorkerProfileRequest,
) (*userv1.CreateWorkerProfileResponse, error) {
	profile, err := s.userService.CreateWorkerProfile(
		ctx,
		req.UserId,
		model.CreateWorkerProfileRequest{
			WorkerType:         req.WorkerType,
			CrewName:           req.CrewName,
			CrewSize:           req.CrewSize,
			ExperienceYears:    req.ExperienceYears,
			ExpectedRate:       req.ExpectedRate,
			RateType:           req.RateType,
			AvailabilityStatus: req.AvailabilityStatus,
			Bio:                req.Bio,
			WorkTypeIDs:        req.WorkTypeIds,
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "22P02", "23503":
				return nil, status.Error(codes.InvalidArgument, "invalid work type id")
			case "23505":
				return nil, status.Error(codes.AlreadyExists, "worker profile already exists")
			}
		}
		return nil, err
	}

	return &userv1.CreateWorkerProfileResponse{
		Profile: toWorkerProfile(profile),
	}, nil
}

func toWorkerProfile(profile *model.WorkerProfile) *userv1.WorkerProfile {
	if profile == nil {
		return nil
	}

	result := &userv1.WorkerProfile{
		Id:                 profile.ID,
		UserId:             profile.UserID,
		WorkerType:         workerTypeEnum(profile.WorkerType),
		CrewName:           profile.CrewName,
		CrewSize:           profile.CrewSize,
		ExperienceYears:    profile.ExperienceYears,
		ExpectedRate:       profile.ExpectedRate,
		RateType:           rateTypeEnum(profile.RateType),
		AvailabilityStatus: profile.AvailabilityStatus,
		VerificationStatus: profile.VerificationStatus,
		Bio:                profile.Bio,
		ProfileCompleted:   profile.ProfileCompleted,
	}
	for _, skill := range profile.Skills {
		result.Skills = append(result.Skills, &userv1.WorkerSkill{SkillId: skill.WorkTypeID})
	}
	return result
}

func workerTypeEnum(value string) userv1.WorkerType {
	switch value {
	case "individual":
		return userv1.WorkerType_WORKER_TYPE_INDIVIDUAL
	case "contractor":
		return userv1.WorkerType_WORKER_TYPE_CONTRACTOR
	default:
		return userv1.WorkerType_WORKER_TYPE_UNSPECIFIED
	}
}

func rateTypeEnum(value *string) userv1.RateType {
	if value == nil {
		return userv1.RateType_RATE_TYPE_UNSPECIFIED
	}
	switch *value {
	case "hourly":
		return userv1.RateType_RATE_TYPE_HOURLY
	case "daily":
		return userv1.RateType_RATE_TYPE_DAILY
	case "fixed":
		return userv1.RateType_RATE_TYPE_FIXED
	default:
		return userv1.RateType_RATE_TYPE_UNSPECIFIED
	}
}

func (s *UserServer) CreateOwnerProfile(
	ctx context.Context,
	req *userv1.CreateOwnerProfileRequest,
) (*userv1.CreateOwnerProfileResponse, error) {

	profile, err := s.userService.CreateOwnerProfile(
		ctx,
		req.UserId,
		model.CreateOwnerProfileRequest{
			RegistrationType:      req.RegistrationType,
			BusinessName:          req.BusinessName,
			BusinessDescription:   req.BusinessDescription,
			BusinessTypeID:        req.BusinessTypeId,
			BusinessSize:          req.BusinessSize,
			GSTRegistrationNumber: req.GstRegistrationNumber,
			FirstName:             req.FirstName,
			LastName:              req.LastName,
			DateOfBirth:           req.DateOfBirth,
			Email:                 req.Email,
			Country:               req.Country,
			State:                 req.State,
			City:                  req.City,
			PostalCode:            req.PostalCode,
			PreferredLanguage:     req.PreferredLanguage,
		},
	)
	if err != nil {
		return nil, err
	}

	return &userv1.CreateOwnerProfileResponse{
		Profile: toOwnerProfile(profile),
	}, nil
}

func (s *UserServer) ListBusinessTypes(
	ctx context.Context,
	_ *userv1.ListBusinessTypesRequest,
) (*userv1.ListBusinessTypesResponse, error) {

	businessTypes, err := s.userService.ListBusinessTypes(ctx)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListBusinessTypesResponse{
		BusinessTypes: make([]*userv1.BusinessType, 0, len(businessTypes)),
	}
	for _, bt := range businessTypes {
		resp.BusinessTypes = append(resp.BusinessTypes, &userv1.BusinessType{
			Id:   bt.ID,
			Name: bt.Name,
		})
	}

	return resp, nil
}

func (s *UserServer) ListWorkTypes(
	ctx context.Context,
	_ *userv1.ListWorkTypesRequest,
) (*userv1.ListWorkTypesResponse, error) {

	workTypes, err := s.userService.ListWorkTypes(ctx)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListWorkTypesResponse{
		WorkTypes: make([]*userv1.WorkType, 0, len(workTypes)),
	}
	for _, wt := range workTypes {
		resp.WorkTypes = append(resp.WorkTypes, &userv1.WorkType{
			Id:   wt.ID,
			Name: wt.Name,
		})
	}

	return resp, nil
}

func toOwnerProfile(profile *model.OwnerProfileResponse) *userv1.OwnerProfile {
	if profile == nil {
		return nil
	}

	pbProfile := &userv1.OwnerProfile{
		ProfileCompleted:  profile.ProfileCompleted,
		UserType:          profile.UserType,
		FirstName:         profile.FirstName,
		LastName:          profile.LastName,
		DateOfBirth:       profile.DateOfBirth,
		Email:             profile.Email,
		Country:           profile.Country,
		State:             profile.State,
		City:              profile.City,
		PostalCode:        profile.PostalCode,
		PreferredLanguage: profile.PreferredLanguage,
	}

	if profile.OwnerProfile != nil {
		pbProfile.OwnerProfile = &userv1.OwnerProfileDetails{
			RegistrationType:      profile.OwnerProfile.RegistrationType,
			BusinessName:          profile.OwnerProfile.BusinessName,
			BusinessDescription:   profile.OwnerProfile.BusinessDescription,
			BusinessSize:          profile.OwnerProfile.BusinessSize,
			GstRegistrationNumber: profile.OwnerProfile.GSTRegistrationNumber,
			BusinessType: &userv1.BusinessType{
				Id:   profile.OwnerProfile.BusinessType.ID,
				Name: profile.OwnerProfile.BusinessType.Name,
			},
		}
	}

	return pbProfile
}

func toUserProfile(profile *model.ProfileResponse) *userv1.UserProfile {
	return &userv1.UserProfile{
		Id:                  profile.ID,
		UserId:              profile.AuthUserID,
		FirstName:           stringValue(profile.FirstName),
		LastName:            stringValue(profile.LastName),
		UserType:            stringValue(profile.UserType),
		ProfileCompleted:    profile.ProfileCompleted,
		OnboardingCompleted: profile.OnboardingCompleted,
		DateOfBirth:         stringValue(profile.DateOfBirth),
		Email:               stringValue(profile.Email),
		Country:             stringValue(profile.Country),
		State:               stringValue(profile.State),
		City:                stringValue(profile.City),
		PostalCode:          stringValue(profile.PostalCode),
		PreferredLanguage:   stringValue(profile.PreferredLanguage),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
