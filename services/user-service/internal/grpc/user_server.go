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
				UserType:   req.UserType,
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

func (s *UserServer) UpdateProfile(
	ctx context.Context,
	req *userv1.UpdateProfileRequest,
) (*userv1.UpdateProfileResponse, error) {
	profile, err := s.userService.UpdateProfile(
		ctx,
		req.UserId,
		model.UpdateProfileRequest{
			FirstName:  req.FirstName,
			LastName:   req.LastName,
			Country:    req.Country,
			State:      req.State,
			City:       req.City,
			PostalCode: req.PostalCode,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProfileNotFound):
			return nil, status.Error(codes.NotFound, "profile not found")
		case errors.Is(err, service.ErrInvalidProfile):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "unable to update profile")
		}
	}

	return &userv1.UpdateProfileResponse{Profile: toUserProfile(profile)}, nil
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
			WorkCategoryID:     req.WorkCategoryId,
			Selections:         workerSelectionsFromProto(req.Selections),
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "22P02", "23503":
				return nil, status.Error(codes.InvalidArgument, "invalid work category, work type, or skill id")
			case "23505":
				return nil, status.Error(codes.AlreadyExists, "worker profile already exists")
			}
		}
		if errors.Is(err, service.ErrInvalidWorkerSelections) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "unable to save worker profile")
	}

	return &userv1.CreateWorkerProfileResponse{
		Profile: toWorkerProfile(profile),
	}, nil
}

func (s *UserServer) UpdateWorkerProfile(
	ctx context.Context,
	req *userv1.UpdateWorkerProfileRequest,
) (*userv1.UpdateWorkerProfileResponse, error) {
	profile, err := s.userService.UpdateWorkerProfile(
		ctx,
		req.UserId,
		model.UpdateWorkerProfileRequest{
			Selections: workerSelectionsFromProto(req.Selections),
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "22P02", "23503":
				return nil, status.Error(codes.InvalidArgument, "invalid work category, work type, or skill id")
			}
		}
		if errors.Is(err, repository.ErrWorkerProfileNotFound) {
			return nil, status.Error(codes.NotFound, "worker profile not found")
		}
		if errors.Is(err, service.ErrInvalidWorkerSelections) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "unable to update worker profile")
	}

	return &userv1.UpdateWorkerProfileResponse{
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
	if profile.WorkCategory != nil {
		result.WorkCategory = &userv1.WorkCategory{
			Id:   profile.WorkCategory.WorkCategoryID,
			Code: profile.WorkCategory.Code,
			Name: profile.WorkCategory.Name,
		}
	}
	result.WorkCategories = workerCategoriesToProto(profile.WorkCategories)
	return result
}

func workerSelectionsFromProto(selections []*userv1.WorkerCategorySelection) []model.WorkerCategorySelection {
	result := make([]model.WorkerCategorySelection, 0, len(selections))
	for _, selection := range selections {
		if selection == nil {
			continue
		}
		workTypes := make([]model.WorkerTypeSelection, 0, len(selection.WorkTypes))
		for _, workType := range selection.WorkTypes {
			if workType == nil {
				continue
			}
			workTypes = append(workTypes, model.WorkerTypeSelection{
				WorkTypeID: workType.WorkTypeId,
				SkillIDs:   workType.SkillIds,
			})
		}
		result = append(result, model.WorkerCategorySelection{
			WorkCategoryID: selection.WorkCategoryId,
			WorkTypes:      workTypes,
		})
	}
	return result
}

func workerCategoriesToProto(categories []model.WorkerWorkCategorySummary) []*userv1.WorkerWorkCategory {
	result := make([]*userv1.WorkerWorkCategory, 0, len(categories))
	for _, category := range categories {
		workTypes := make([]*userv1.WorkerWorkType, 0, len(category.WorkTypes))
		for _, workType := range category.WorkTypes {
			skills := make([]*userv1.WorkerSkill, 0, len(workType.Skills))
			for _, skill := range workType.Skills {
				skills = append(skills, &userv1.WorkerSkill{
					Id:         skill.ID,
					WorkTypeId: skill.WorkTypeID,
					Code:       skill.Code,
					Name:       skill.Name,
				})
			}
			workTypes = append(workTypes, &userv1.WorkerWorkType{
				WorkTypeId: workType.WorkTypeID,
				Name:       workType.Name,
				CategoryId: workType.CategoryID,
				IsPrimary:  workType.IsPrimary,
				Skills:     skills,
			})
		}
		result = append(result, &userv1.WorkerWorkCategory{
			WorkCategoryId: category.WorkCategoryID,
			Code:           category.Code,
			Name:           category.Name,
			WorkTypes:      workTypes,
		})
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
		resp.WorkTypes = append(resp.WorkTypes, toWorkTypeProto(wt))
	}

	return resp, nil
}

func (s *UserServer) ListWorkTypesByCategory(
	ctx context.Context,
	req *userv1.ListWorkTypesByCategoryRequest,
) (*userv1.ListWorkTypesResponse, error) {

	workTypes, err := s.userService.ListWorkTypesByCategory(ctx, req.CategoryId)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListWorkTypesResponse{
		WorkTypes: make([]*userv1.WorkType, 0, len(workTypes)),
	}
	for _, wt := range workTypes {
		resp.WorkTypes = append(resp.WorkTypes, toWorkTypeProto(wt))
	}

	return resp, nil
}

func toWorkTypeProto(wt model.WorkType) *userv1.WorkType {
	return &userv1.WorkType{
		Id:           wt.ID,
		Name:         wt.Name,
		CategoryId:   wt.CategoryID,
		Code:         wt.Code,
		Description:  wt.Description,
		DisplayOrder: wt.DisplayOrder,
		IsActive:     wt.IsActive,
	}
}

func (s *UserServer) ListWorkTypeFields(
	ctx context.Context,
	req *userv1.ListWorkTypeFieldsRequest,
) (*userv1.ListWorkTypeFieldsResponse, error) {

	fields, err := s.userService.ListWorkTypeFields(ctx, req.WorkTypeId)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListWorkTypeFieldsResponse{
		Fields: make([]*userv1.WorkTypeField, 0, len(fields)),
	}
	for _, f := range fields {
		resp.Fields = append(resp.Fields, toWorkTypeFieldProto(f))
	}

	return resp, nil
}

func toWorkTypeFieldProto(f model.WorkTypeField) *userv1.WorkTypeField {
	pbField := &userv1.WorkTypeField{
		Id:           f.ID,
		WorkTypeId:   f.WorkTypeID,
		FieldKey:     f.FieldKey,
		Label:        f.Label,
		FieldType:    f.FieldType,
		Placeholder:  f.Placeholder,
		HelpText:     f.HelpText,
		IsRequired:   f.IsRequired,
		Unit:         f.Unit,
		MinValue:     f.MinValue,
		MaxValue:     f.MaxValue,
		MinLength:    f.MinLength,
		MaxLength:    f.MaxLength,
		DisplayOrder: f.DisplayOrder,
		IsActive:     f.IsActive,
		Options:      make([]*userv1.WorkTypeFieldOption, 0, len(f.Options)),
	}
	for _, opt := range f.Options {
		pbField.Options = append(pbField.Options, &userv1.WorkTypeFieldOption{
			Id:           opt.ID,
			Value:        opt.Value,
			Label:        opt.Label,
			DisplayOrder: opt.DisplayOrder,
			IsActive:     opt.IsActive,
			FieldKey:     opt.FieldKey,
		})
	}
	return pbField
}

func (s *UserServer) ListWorkTypePaymentTypes(
	ctx context.Context,
	req *userv1.ListWorkTypePaymentTypesRequest,
) (*userv1.ListPaymentTypesResponse, error) {
	paymentTypes, err := s.userService.ListWorkTypePaymentTypes(ctx, req.WorkTypeId)
	if err != nil {
		return nil, err
	}
	return toPaymentTypesResponse(paymentTypes), nil
}

func (s *UserServer) ListWorkCategoryPaymentTypes(
	ctx context.Context,
	req *userv1.ListWorkCategoryPaymentTypesRequest,
) (*userv1.ListPaymentTypesResponse, error) {
	paymentTypes, err := s.userService.ListWorkCategoryPaymentTypes(ctx, req.CategoryId)
	if err != nil {
		return nil, err
	}
	return toPaymentTypesResponse(paymentTypes), nil
}

func toPaymentTypesResponse(paymentTypes []model.PaymentType) *userv1.ListPaymentTypesResponse {
	resp := &userv1.ListPaymentTypesResponse{
		PaymentTypes: make([]*userv1.PaymentType, 0, len(paymentTypes)),
	}
	for _, paymentType := range paymentTypes {
		resp.PaymentTypes = append(resp.PaymentTypes, &userv1.PaymentType{
			Id:        paymentType.ID,
			Code:      paymentType.Code,
			Name:      paymentType.Name,
			IsActive:  paymentType.IsActive,
			CreatedAt: paymentType.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return resp
}

func (s *UserServer) ListWorkTypeSkills(
	ctx context.Context,
	req *userv1.ListWorkTypeSkillsRequest,
) (*userv1.ListWorkTypeSkillsResponse, error) {
	skills, err := s.userService.ListWorkTypeSkills(ctx, req.WorkTypeId)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListWorkTypeSkillsResponse{
		Skills: make([]*userv1.Skill, 0, len(skills)),
	}
	for _, skill := range skills {
		resp.Skills = append(resp.Skills, &userv1.Skill{
			Id:         skill.ID,
			WorkTypeId: skill.WorkTypeID,
			Code:       skill.Code,
			Name:       skill.Name,
			IsActive:   skill.IsActive,
			CreatedAt:  skill.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:  skill.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return resp, nil
}

func (s *UserServer) ListWorkCategories(
	ctx context.Context,
	_ *userv1.ListWorkCategoriesRequest,
) (*userv1.ListWorkCategoriesResponse, error) {

	categories, err := s.userService.ListWorkCategories(ctx)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListWorkCategoriesResponse{
		Categories: make([]*userv1.WorkCategory, 0, len(categories)),
	}
	for _, wc := range categories {
		resp.Categories = append(resp.Categories, &userv1.WorkCategory{
			Id:           wc.ID,
			Code:         wc.Code,
			Name:         wc.Name,
			Description:  wc.Description,
			DisplayOrder: wc.DisplayOrder,
			IsActive:     wc.IsActive,
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
	if profile == nil {
		return nil
	}

	result := &userv1.UserProfile{
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
		WorkCategories:      workerCategoriesToProto(profile.WorkCategories),
	}
	if profile.WorkCategory != nil {
		result.WorkCategory = &userv1.WorkCategory{
			Id:   profile.WorkCategory.WorkCategoryID,
			Code: profile.WorkCategory.Code,
			Name: profile.WorkCategory.Name,
		}
	}
	return result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
