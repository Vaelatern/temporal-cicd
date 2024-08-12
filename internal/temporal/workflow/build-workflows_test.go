package workflow

import (
	"errors"
	"testing"

	"github.com/Vaelatern/temporal-cicd/internal/temporal/activity"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *UnitTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *UnitTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestWorkflows(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func TestMakeBuildWorkflow(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func (s *UnitTestSuite) Test_MakeBuild_ActivityFails() {
	s.env.OnActivity(activity.MakeBuild, mock.Anything, mock.Anything).Return(activity.BuildResponse{}, errors.New("Incorrect Function"))
	s.env.ExecuteWorkflow(MakeBuildWorkflow, activity.BuildDetails{})

	s.True(s.env.IsWorkflowCompleted())

	err := s.env.GetWorkflowError()
	s.Error(err)
	var applicationErr *temporal.ApplicationError
	s.True(errors.As(err, &applicationErr))
	s.Equal("Incorrect Function", applicationErr.Error())
}
