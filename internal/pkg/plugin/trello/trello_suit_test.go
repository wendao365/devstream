package trello_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlanmanager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Trello Test Suite")
}
