package git_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/devstream-io/devstream/pkg/util/git"
)

type mockRepoStruct struct {
	initRaiseError  bool
	pushRaiseError  bool
	needRollBack    bool
	deleteFuncIsRun bool
}

func (m *mockRepoStruct) InitRepo() error {
	if m.initRaiseError {
		return errors.New("init error")
	}
	return nil
}

func (m *mockRepoStruct) PushLocalFileToRepo(commitInfo *git.CommitInfo) (bool, error) {
	if m.pushRaiseError {
		return m.needRollBack, errors.New("push error")
	}
	return m.needRollBack, nil
}

func (m *mockRepoStruct) DeleteRepo() error {
	m.deleteFuncIsRun = true
	return nil
}

var _ = Describe("PushInitRepo func", func() {
	var (
		mockRepo   *mockRepoStruct
		commitInfo *git.CommitInfo
		err        error
	)
	BeforeEach(func() {
		commitInfo = &git.CommitInfo{
			CommitMsg:    "test",
			CommitBranch: "test-branch",
		}
	})
	When("init method return err", func() {
		BeforeEach(func() {
			mockRepo = &mockRepoStruct{
				initRaiseError: true,
				pushRaiseError: false,
			}
		})
		It("should return err", func() {
			err = git.PushInitRepo(mockRepo, commitInfo)
			Expect(err).Error().Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("init error"))
		})
	})

	When("push method failed", func() {
		BeforeEach(func() {
			mockRepo = &mockRepoStruct{
				initRaiseError: false,
				pushRaiseError: true,
				needRollBack:   false,
			}
		})
		It("should return err", func() {
			err = git.PushInitRepo(mockRepo, commitInfo)
			Expect(err).Error().Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("push error"))
			Expect(mockRepo.deleteFuncIsRun).Should(BeFalse())
		})

		When("push method return needRollBack", func() {
			BeforeEach(func() {
				mockRepo = &mockRepoStruct{
					initRaiseError: false,
					pushRaiseError: true,
					needRollBack:   true,
				}
			})
			It("should run DeleteRepo method", func() {
				err = git.PushInitRepo(mockRepo, commitInfo)
				Expect(err).Error().Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("push error"))
				Expect(mockRepo.deleteFuncIsRun).Should(BeTrue())
			})
		})
	})
})
