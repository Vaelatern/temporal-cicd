package workflow

import (
	"github.com/Vaelatern/temporal-cicd/internal/temporal/activity"
	"github.com/stretchr/testify/mock"
)

func (s *UnitTestSuite) Test_BuildLifecycleWorkflowGoldenPath() {
	s.env.OnActivity(activity.AddRowToSmartsheet, mock.Anything, mock.Anything).Return(activity.SmartSheetTaskResponse{}, nil)
	s.env.OnActivity(activity.SetDetailsInSmartsheet, mock.Anything, mock.Anything).Return(activity.SmartSheetTaskResponse{}, nil)
	s.env.OnActivity(activity.MakeBuild, mock.Anything, mock.Anything).Return(activity.BuildResponse{}, nil)
	s.env.OnActivity(activity.UploadBuildResultsToSmartsheet, mock.Anything, mock.Anything).Return(activity.SmartSheetTaskResponse{}, nil)
	s.env.OnActivity(activity.DeclareBuildReadySmartsheet, mock.Anything, mock.Anything).Return(activity.SmartSheetTaskResponse{}, nil)
	s.env.OnActivity(activity.DeployToEnv, mock.Anything, mock.Anything).Return(activity.DeployEnvResponse{}, nil)
	s.env.OnActivity(activity.DeleteRowFromSmartsheet, mock.Anything, mock.Anything).Return(activity.SmartSheetTaskResponse{}, nil)

	//s.env.OnActivity(activity.SmartSheetNotify, mock.Anything, mock.Anything).Return(activity.SmartSheetTaskResponse{}, nil)
	//s.env.OnActivity(activity.MarkRowTamperedSmartsheet, mock.Anything, mock.Anything).Return(activity.SmartSheetTaskResponse{}, nil)

	s.env.RegisterWorkflow(MakeBuildWorkflow)

	s.env.ExecuteWorkflow(BuildLifecycleWorkflow)

	s.True(s.env.IsWorkflowCompleted())

	//err := s.env.GetWorkflowError()
	//s.Error(err)
	//var applicationErr *temporal.ApplicationError
	//s.True(errors.As(err, &applicationErr))
	//s.Equal("SimpleActivityFailure", applicationErr.Error())
}
