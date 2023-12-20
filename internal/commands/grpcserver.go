package commands

import (
	"commandos/internal/appconst"
	"commandos/internal/services/logger"
	"commandos/internal/types"
	"commandos/internal/validator"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	driver "gitlab.kvant.online/seal/driver/670"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	drvErrors "gitlab.kvant.online/seal/driver/errors"
	commands_v1 "gitlab.kvant.online/seal/grpc-contracts/pkg/commands/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AppError struct {
	Body     any   `json:"message,omitempty"`
	httpCode int   `json:"-"`
	Err      error `json:"err,omitempty"`
}

func (e AppError) Error() string {
	return fmt.Sprintf("%v", e.Body)
}

type repo interface {
	AddCommand(imei string, name string, params string, author string) (int32, string, error)
	ListCommands(imei string, offset int32, limit int32, newOnly bool) ([]types.Command, error)
	UpdCommand(id int32, responseDate time.Time, response, rawRequest, rawResponse string) (int32, int64, error)
	UpdTry(id int32) error
}

// GRPCServer ...
type GRPCServer struct {
	repo   repo
	logger *logger.Logger
	commands_v1.UnimplementedCommandsServiceServer
}

func NewGRPCServer(repo repo, logger *logger.Logger) *GRPCServer {
	return &GRPCServer{
		logger: logger,
		repo:   repo,
	}
}

// Add command...
func (s *GRPCServer) Add(ctx context.Context, req *commands_v1.AddRequest) (*commands_v1.AddUpdResponse, error) {

	driverValidator, cmdErr := driver.NewCommander(req.GetProtocol())

	if cmdErr != nil {
		return nil, cmdErr
	}

	validateErrors, vldErr := driverValidator.ValidateParams(req.Name, []byte(req.Params))

	if vldErr != nil {
		return nil, vldErr
	}

	if !validator.StrIsValid(req.Imei, 15, 16) {
		validateErrors.AddError(drvErrors.ErrValidation{Field: "imei", Message: "IMEI должен быть от 15 до 16 цифр"})
	}

	if !validator.StrIsValid(req.Author, 3, 30) {
		validateErrors.AddError(drvErrors.ErrValidation{Field: "author", Message: "Имя автора должно быть от 3 до 30 символов"})
	}

	if !validateErrors.Success() {

		st := status.New(codes.InvalidArgument, "Invalid argument")
		br := &errdetails.BadRequest{}

		for _, vld := range validateErrors.Errors() {
			log.Println(vld.Field, vld.Message)
			v := &errdetails.BadRequest_FieldViolation{
				Field:       vld.Field,
				Description: vld.Message,
			}
			br.FieldViolations = append(br.FieldViolations, v)
		}

		st, err := st.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("Unexpected error attaching metadata: %v", err))
		}
		return nil, st.Err()
	}

	code, message, err := s.repo.AddCommand(req.Imei, req.Name, req.Params, req.Author)

	if err != nil {
		s.logger.Error.Println(err)
	}

	if code == appconst.DuplicateCommand {
		return nil, status.Error(codes.AlreadyExists, "Команда уже находится в очереди")
	}

	if code > 0 {
		return &commands_v1.AddUpdResponse{Code: code, Messages: map[string]string{"db": message}}, err
	}

	return &commands_v1.AddUpdResponse{Code: 0, Messages: nil}, nil
}

// List of commands
func (s *GRPCServer) List(ctx context.Context, req *commands_v1.ListRequest) (*commands_v1.ListResponse, error) {

	list, err := s.repo.ListCommands(req.Imei, req.Offset, req.Limit, req.NewOnly)
	if err != nil {
		s.logger.Error.Println(err)
		return &commands_v1.ListResponse{Commands: nil}, err
	}

	commands := make([]*commands_v1.Command, 0)

	for _, item := range list {

		var tryDate, responseDate, abortDate *timestamppb.Timestamp

		if item.TryDate != nil {
			tryDate = timestamppb.New(*item.TryDate)
		}
		if item.ResponseDate != nil {
			responseDate = timestamppb.New(*item.ResponseDate)
		}

		if item.AbortDate != nil {
			abortDate = timestamppb.New(*item.AbortDate)
		}

		commands = append(commands, &commands_v1.Command{
			Id:           *item.Id,
			Imei:         item.Imei,
			Name:         item.Name,
			Params:       item.Params,
			Author:       item.Author,
			Dateon:       timestamppb.New(item.Dateon),
			TryNumber:    item.TryNumber,
			TryDate:      tryDate,
			ResponseDate: responseDate,
			Response:     item.Response,
			RawRequest:   item.RawRequest,
			RawResponse:  item.RawResponse,
			AbortDate:    abortDate,
		})
	}

	return &commands_v1.ListResponse{Commands: commands}, nil
}

// Get command
func (s *GRPCServer) Get(ctx context.Context, req *commands_v1.GetRequest) (*commands_v1.GetResponse, error) {

	list, err := s.repo.ListCommands(req.Imei, 0, 1, true)

	if err != nil {
		s.logger.Error.Println(err)
		return nil, err
	}

	for _, item := range list {
		err := s.repo.UpdTry(*item.Id)
		if err != nil {
			return nil, err
		}
		return &commands_v1.GetResponse{Id: *item.Id, Name: item.Name, Params: item.Params}, nil
	}

	return nil, status.Error(codes.NotFound, fmt.Sprintf(`Command for imei %s not found`, req.Imei))
}

// Upd command...
func (s *GRPCServer) Upd(ctx context.Context, req *commands_v1.UpdRequest) (*commands_v1.AddUpdResponse, error) {

	errs := map[string]string{}

	//if !validator.StrIsValid(req.GetResponse(), 3, 30) {
	//	errors["responce"] = "Серийный номер должен быть от 3 до 30 символов"
	//}

	if len(errs) > 0 {
		return &commands_v1.AddUpdResponse{Code: appconst.InvalidInputData, Messages: errs}, nil
	}

	loc, tzErr := time.LoadLocation("Europe/Moscow")

	if tzErr != nil {
		s.logger.Error.Println(tzErr)
	}

	code, rowsAffected, err := s.repo.UpdCommand(req.GetId(), req.GetResponseDate().AsTime().In(loc), req.GetResponse(), req.GetRawRequest(), req.GetRawResponse())

	if err != nil {
		s.logger.Error.Println(err)
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, errors.New("record not found")
	}

	if code > 0 {
		return nil, err
	}

	message := fmt.Sprintf(`rowsAffected: %d`, rowsAffected)

	return &commands_v1.AddUpdResponse{Code: code, Messages: map[string]string{"db": message}}, nil
}
