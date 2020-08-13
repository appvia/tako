package kev_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestKev(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kev Suite")
}
