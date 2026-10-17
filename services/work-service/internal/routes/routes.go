package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/middleware"
)

func RegisterRoutes(
	router *gin.Engine,
	workHandler *handler.WorkHandler,
	jwtSecret string,
) {
	router.GET("/health", workHandler.Health)

	v1 := router.Group("/api/v1")
	{
		work := v1.Group("/work")
		work.Use(middleware.AuthMiddleware(jwtSecret))
		{
			work.POST("", workHandler.CreateWork)
			work.GET("", workHandler.ListMyWorks)
			work.GET("/applications", workHandler.ListMyApplications)
			work.POST("/:workId/applications", workHandler.ApplyToWork)
			work.GET("/:workId/applications", workHandler.ListWorkApplications)
			work.POST("/:workId/applications/:applicationId/shortlist", workHandler.ShortlistWorkApplication)
			work.POST("/:workId/applications/:applicationId/accept", workHandler.AcceptWorkApplication)
			work.POST("/:workId/applications/:applicationId/reject", workHandler.RejectWorkApplication)
			work.POST("/:workId/applications/:applicationId/withdraw", workHandler.WithdrawWorkApplication)
			work.POST("/:workId/applications/:applicationId/cancel", workHandler.CancelWorkApplication)
			work.POST("/:workId/start", workHandler.StartWork)
			work.POST("/:workId/complete", workHandler.CompleteWork)
			work.POST("/:workId/confirm-completion", workHandler.ConfirmWorkCompletion)
			work.POST("/:workId/complete-job", workHandler.CompleteJob)
			work.POST("/:workId/close-applications", workHandler.CloseApplications)
			work.POST("/:workId/close", workHandler.CloseWork)
			work.POST("/:workId/cancel", workHandler.CancelWork)
			work.GET("/:workId/daily-logs", workHandler.ListWorkDailyLogs)
			work.GET("/:workId/settlement", workHandler.GetWorkSettlement)
			work.POST("/:workId/assignments/:assignmentId/payments", workHandler.RecordAssignmentPayment)
			work.GET("/:workId/work-records/summary", workHandler.GetWorkerWorkSummary)
			work.GET("/:workId/work-records", workHandler.GetWorkDayRecords)
			work.POST("/:workId/work-records", workHandler.SaveWorkDayRecords)
			work.GET("/:workId/contractor-work-records/summary", workHandler.GetContractorWorkSummary)
			work.GET("/:workId/contractor-work-records", workHandler.GetContractorWorkRecords)
			work.POST("/:workId/contractor-work-records", workHandler.SaveContractorWorkRecord)
			work.GET("/:workId/payments", workHandler.GetWorkPayments)
		}

		works := v1.Group("/works")
		works.Use(middleware.AuthMiddleware(jwtSecret))
		{
			works.POST("/:workId/start", workHandler.StartWork)
			works.POST("/:workId/complete", workHandler.CompleteWork)
			works.POST("/:workId/close", workHandler.CloseWork)
			works.GET("/:workId/work-records/summary", workHandler.GetWorkerWorkSummary)
			works.GET("/:workId/work-records", workHandler.GetWorkDayRecords)
			works.POST("/:workId/work-records", workHandler.SaveWorkDayRecords)
			works.GET("/:workId/contractor-work-records/summary", workHandler.GetContractorWorkSummary)
			works.GET("/:workId/contractor-work-records", workHandler.GetContractorWorkRecords)
			works.POST("/:workId/contractor-work-records", workHandler.SaveContractorWorkRecord)
			works.GET("/:workId/payments", workHandler.GetWorkPayments)
		}

		obligations := v1.Group("/payment-obligations")
		obligations.Use(middleware.AuthMiddleware(jwtSecret))
		{
			obligations.POST("/:paymentId/record-payment", workHandler.RecordObligationPayment)
		}

		assignments := v1.Group("/work-assignments")
		assignments.Use(middleware.AuthMiddleware(jwtSecret))
		{
			assignments.POST("/:assignmentId/start", workHandler.StartAssignment)
			assignments.POST("/:assignmentId/complete", workHandler.CompleteAssignment)
			assignments.PUT("/:assignmentId/daily-logs/:date", workHandler.UpsertDailyLog)
			assignments.GET("/:assignmentId/daily-logs", workHandler.ListAssignmentDailyLogs)
			assignments.GET("/:assignmentId/payments", workHandler.ListAssignmentPayments)
			assignments.POST("/:assignmentId/payments/:paymentId/confirm", workHandler.ConfirmAssignmentPayment)
			assignments.POST("/:assignmentId/payments/:paymentId/dispute", workHandler.DisputeAssignmentPayment)
		}
	}
}
