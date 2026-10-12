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
				Gender:     req.Gender,
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
		if errors.Is(err, service.ErrInvalidProfile) || errors.Is(err, service.ErrInvalidGender) {
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
			Gender:     req.Gender,
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
		case errors.Is(err, service.ErrInvalidProfile), errors.Is(err, service.ErrInvalidGender):
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
			ExpectedRateMin:    req.ExpectedRateMin,
			ExpectedRateMax:    req.ExpectedRateMax,
			RateBasis:          req.RateBasis,
			RateUnit:           req.RateUnit,
			AvailabilityStatus: req.AvailabilityStatus,
			Bio:                req.Bio,
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
			RateBasis:  req.RateBasis,
			RateUnit:   req.RateUnit,
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
		ExpectedRateMin:    profile.ExpectedRateMin,
		ExpectedRateMax:    profile.ExpectedRateMax,
		RateBasis:          profile.RateBasis,
		RateUnit:           profile.RateUnit,
		AvailabilityStatus: profile.AvailabilityStatus,
		VerificationStatus: profile.VerificationStatus,
		Bio:                profile.Bio,
		ProfileCompleted:   profile.ProfileCompleted,
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
		segments := make([]model.WorkerSegmentSelection, 0, len(selection.Segments))
		for _, segment := range selection.Segments {
			if segment == nil {
				continue
			}
			segments = append(segments, model.WorkerSegmentSelection{
				WorkSegmentID: segment.WorkSegmentId,
				WorkTypes:     workerTypeSelectionsFromProto(segment.WorkTypes),
			})
		}
		if len(segments) == 0 && len(selection.WorkTypes) > 0 {
			segments = append(segments, model.WorkerSegmentSelection{
				WorkTypes: workerTypeSelectionsFromProto(selection.WorkTypes),
			})
		}
		result = append(result, model.WorkerCategorySelection{
			WorkCategoryID: selection.WorkCategoryId,
			Segments:       segments,
		})
	}
	return result
}

func workerTypeSelectionsFromProto(workTypes []*userv1.WorkerTypeSelection) []model.WorkerTypeSelection {
	result := make([]model.WorkerTypeSelection, 0, len(workTypes))
	for _, workType := range workTypes {
		if workType == nil {
			continue
		}
		result = append(result, model.WorkerTypeSelection{
			WorkTypeID: workType.WorkTypeId,
			SkillIDs:   workType.SkillIds,
		})
	}
	return result
}

func workerWorkTypesToProto(workTypes []model.WorkerWorkTypeSummary) []*userv1.WorkerWorkType {
	result := make([]*userv1.WorkerWorkType, 0, len(workTypes))
	for _, workType := range workTypes {
		skills := make([]*userv1.WorkerSkill, 0, len(workType.Skills))
		for _, skill := range workType.Skills {
			skills = append(skills, &userv1.WorkerSkill{
				Id:         skill.ID,
				WorkTypeId: skill.WorkTypeID,
				Code:       skill.Code,
				Name:       skill.Name,
			})
		}
		result = append(result, &userv1.WorkerWorkType{
			WorkTypeId: workType.WorkTypeID,
			Name:       workType.Name,
			CategoryId: workType.CategoryID,
			SegmentId:  workType.SegmentID,
			IsPrimary:  workType.IsPrimary,
			Skills:     skills,
		})
	}
	return result
}

func workerCategoriesToProto(categories []model.WorkerWorkCategorySummary) []*userv1.WorkerWorkCategory {
	result := make([]*userv1.WorkerWorkCategory, 0, len(categories))
	for _, category := range categories {
		segments := make([]*userv1.WorkerWorkSegment, 0, len(category.Segments))
		for _, segment := range category.Segments {
			segments = append(segments, &userv1.WorkerWorkSegment{
				WorkSegmentId: segment.WorkSegmentID,
				Code:          segment.Code,
				Name:          segment.Name,
				CategoryId:    segment.CategoryID,
				WorkTypes:     workerWorkTypesToProto(segment.WorkTypes),
			})
		}
		result = append(result, &userv1.WorkerWorkCategory{
			WorkCategoryId: category.WorkCategoryID,
			Code:           category.Code,
			Name:           category.Name,
			WorkTypes:      workerWorkTypesToProto(category.WorkTypes),
			Segments:       segments,
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
			Gender:                req.Gender,
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
		if errors.Is(err, service.ErrInvalidGender) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
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
		SegmentId:    wt.SegmentID,
		Code:         wt.Code,
		Description:  wt.Description,
		DisplayOrder: wt.DisplayOrder,
		IsActive:     wt.IsActive,
	}
}

func (s *UserServer) ListWorkSegmentsByCategory(
	ctx context.Context,
	req *userv1.ListWorkSegmentsByCategoryRequest,
) (*userv1.ListWorkSegmentsResponse, error) {
	segments, err := s.userService.ListWorkSegmentsByCategory(ctx, req.CategoryId)
	if err != nil {
		return nil, err
	}

	resp := &userv1.ListWorkSegmentsResponse{
		Segments: make([]*userv1.WorkSegment, 0, len(segments)),
	}
	for _, segment := range segments {
		resp.Segments = append(resp.Segments, &userv1.WorkSegment{
			Id:           segment.ID,
			CategoryId:   segment.CategoryID,
			Code:         segment.Code,
			Name:         segment.Name,
			Description:  segment.Description,
			DisplayOrder: segment.DisplayOrder,
			IsActive:     segment.IsActive,
		})
	}
	return resp, nil
}

func (s *UserServer) ListWorkTypesBySegment(
	ctx context.Context,
	req *userv1.ListWorkTypesBySegmentRequest,
) (*userv1.ListWorkTypesResponse, error) {
	workTypes, err := s.userService.ListWorkTypesBySegment(ctx, req.SegmentId)
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
		Context:      f.Context,
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

func (s *UserServer) ListWorkTypeRateTypes(
	ctx context.Context,
	req *userv1.ListWorkTypeRateTypesRequest,
) (*userv1.ListRateTypesResponse, error) {
	options, err := s.userService.ListWorkTypeRateTypes(ctx, req.WorkTypeId)
	if err != nil {
		return nil, err
	}
	return toRateTypesResponse(options), nil
}

func (s *UserServer) ListWorkSegmentRateTypes(
	ctx context.Context,
	req *userv1.ListWorkSegmentRateTypesRequest,
) (*userv1.ListRateTypesResponse, error) {
	options, err := s.userService.ListWorkSegmentRateTypes(ctx, req.SegmentId)
	if err != nil {
		return nil, err
	}
	return toRateTypesResponse(options), nil
}

func (s *UserServer) ListWorkCategoryRateTypes(
	ctx context.Context,
	req *userv1.ListWorkCategoryRateTypesRequest,
) (*userv1.ListRateTypesResponse, error) {
	options, err := s.userService.ListWorkCategoryRateTypes(ctx, req.CategoryId)
	if err != nil {
		return nil, err
	}
	return toRateTypesResponse(options), nil
}

func toRateTypesResponse(options []model.RateOption) *userv1.ListRateTypesResponse {
	resp := &userv1.ListRateTypesResponse{
		RateTypes: make([]*userv1.RateOption, 0, len(options)),
	}
	for _, option := range options {
		resp.RateTypes = append(resp.RateTypes, &userv1.RateOption{
			RateBasis:     option.RateBasis,
			RateBasisName: option.RateBasisName,
			RateUnit:      option.RateUnit,
			RateUnitName:  option.RateUnitName,
			IsDefault:     option.IsDefault,
			Label:         option.Label,
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
		resp.Skills = append(resp.Skills, toSkillProto(skill))
	}
	return resp, nil
}

func toSkillProto(skill model.Skill) *userv1.Skill {
	return &userv1.Skill{
		Id:         skill.ID,
		Code:       skill.Code,
		Name:       skill.Name,
		IsActive:   skill.IsActive,
		CreatedAt:  skill.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  skill.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsRequired: skill.IsRequired,
	}
}

func toCommonRoleProto(role model.CommonRole) *userv1.CommonRole {
	return &userv1.CommonRole{
		Id:        role.ID,
		Code:      role.Code,
		Name:      role.Name,
		IsActive:  role.IsActive,
		CreatedAt: role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsPrimary: role.IsPrimary,
	}
}

func (s *UserServer) ListWorkTypeCommonRoles(
	ctx context.Context,
	req *userv1.ListWorkTypeCommonRolesRequest,
) (*userv1.ListWorkTypeCommonRolesResponse, error) {
	roles, err := s.userService.ListWorkTypeCommonRoles(ctx, req.WorkTypeId)
	if err != nil {
		return nil, err
	}
	resp := &userv1.ListWorkTypeCommonRolesResponse{
		CommonRoles: make([]*userv1.CommonRole, 0, len(roles)),
	}
	for _, role := range roles {
		resp.CommonRoles = append(resp.CommonRoles, toCommonRoleProto(role))
	}
	return resp, nil
}

func (s *UserServer) GetWorkTypeConfiguration(
	ctx context.Context,
	req *userv1.GetWorkTypeConfigurationRequest,
) (*userv1.GetWorkTypeConfigurationResponse, error) {
	config, err := s.userService.GetWorkTypeConfiguration(ctx, req.WorkTypeId)
	if err != nil {
		return nil, err
	}

	resp := &userv1.GetWorkTypeConfigurationResponse{
		Skills:      make([]*userv1.Skill, 0, len(config.Skills)),
		CommonRoles: make([]*userv1.CommonRole, 0, len(config.CommonRoles)),
		Fields:      make([]*userv1.WorkTypeField, 0, len(config.Fields)),
	}
	for _, skill := range config.Skills {
		resp.Skills = append(resp.Skills, toSkillProto(skill))
	}
	for _, role := range config.CommonRoles {
		resp.CommonRoles = append(resp.CommonRoles, toCommonRoleProto(role))
	}
	for _, field := range config.Fields {
		resp.Fields = append(resp.Fields, toWorkTypeFieldProto(field))
	}
	resp.RateTypes = toRateTypesResponse(config.RateTypes).RateTypes
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
		Gender:            profile.Gender,
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
		Gender:              profile.Gender,
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
		ExpectedRateMin:     profile.ExpectedRateMin,
		ExpectedRateMax:     profile.ExpectedRateMax,
		RateBasis:           profile.RateBasis,
		RateUnit:            profile.RateUnit,
	}
	return result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
