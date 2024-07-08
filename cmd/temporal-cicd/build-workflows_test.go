package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
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

func TestPodmanBuildWorkflow(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func TestMakeBuildWorkflow(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}
