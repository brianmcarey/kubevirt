/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virtlauncher

import (
	"flag"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
)

var fakeQEMUBinary string

func init() {
	flag.StringVar(&fakeQEMUBinary, "fake-qemu-binary-path", "_out/cmd/fake-qemu-process/fake-qemu-process", "path to cirros test image")
}

var _ = Describe("VirtLauncher", func() {
	var mon *monitor
	var cmd *exec.Cmd
	var cmdLock sync.Mutex
	var gracefulShutdownChannel chan struct{}
	var fakeUuid string
	var pidDir string
	var processStarted bool
	var err error

	StartProcess := func() {
		cmdLock.Lock()
		defer cmdLock.Unlock()

		cmd = exec.Command(fakeQEMUBinary, "--uuid", fakeUuid, "--pidfile", filepath.Join(pidDir, "fakens_fakevmi.pid"))
		err = cmd.Start()
		Expect(err).ToNot(HaveOccurred())

		currentPid := cmd.Process.Pid
		Expect(currentPid).ToNot(Equal(0))

		processStarted = true
	}

	StopProcess := func() {
		cmdLock.Lock()
		defer cmdLock.Unlock()

		cmd.Process.Kill()

		processStarted = false
	}

	CleanupProcess := func() {
		cmdLock.Lock()
		defer cmdLock.Unlock()

		cmd.Wait()
	}

	VerifyProcessStarted := func() {
		Eventually(func() bool {

			mon.refresh()
			if mon.pid != 0 {
				return true
			}
			return false

		}).Should(BeTrue())

	}

	VerifyProcessStopped := func() {
		Eventually(func() bool {

			mon.refresh()
			if mon.pid == 0 && mon.isDone == true {
				return true
			}
			return false

		}).Should(BeTrue())

	}

	BeforeEach(func() {
		fakeUuid = uuid.New().String()
		pidDir = GinkgoT().TempDir()

		processStarted = false
		if !strings.Contains(fakeQEMUBinary, "../../") {
			fakeQEMUBinary = filepath.Join("../../", fakeQEMUBinary)
		}
		gracefulShutdownChannel = make(chan struct{})
		shutdownCallback := func(pid int) {
			// Don't send SIGTERM to the current process group (i.e. to PID 0). That will interrupt
			// the test run.
			Expect(pid).ToNot(BeZero())
			err := syscall.Kill(pid, syscall.SIGTERM)
			Expect(err).ToNot(HaveOccurred())
		}
		gracefulShutdownCallback := func() {
			close(gracefulShutdownChannel)
		}
		mon = &monitor{
			domainName:               "fakens_fakevmi",
			pidDir:                   pidDir,
			gracePeriod:              30,
			finalShutdownCallback:    shutdownCallback,
			gracefulShutdownCallback: gracefulShutdownCallback,
		}
	})

	AfterEach(func() {
		if processStarted == true {
			StopProcess()
			CleanupProcess()
		}
	})

	Describe("VirtLauncher", func() {
		Context("process monitor", func() {
			It("verify pid detection works", func() {
				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()
				StopProcess()
				VerifyProcessStopped()
			})

			It("verify zombie pid detection works", func() {
				StartProcess()
				VerifyProcessStarted()
				StopProcess()
				VerifyProcessStopped()

				// cleanup after stopping ensures zombie process is detected
				CleanupProcess()
			})

			It("verify start timeout works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				go func() {
					defer GinkgoRecover()
					mon.RunForever(time.Second, stopChan)
					done <- "exit"
				}()
				noExitCheck := time.After(3 * time.Second)

				exited := false
				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(BeTrue())
			})

			It("verify monitor loop exits when signal arrives and no pid is present", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				go func() {
					defer GinkgoRecover()
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				time.Sleep(time.Second)

				close(stopChan)
				noExitCheck := time.After(5 * time.Second)
				exited := false

				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(BeTrue())
			})

			It("verify graceful shutdown trigger works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()

				go func() {
					defer GinkgoRecover()
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				Consistently(gracefulShutdownChannel).Should(Not(BeClosed()))
				close(stopChan)
				Eventually(gracefulShutdownChannel).Should(BeClosed())
			})

			It("verify grace period works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()
				go func() {
					defer GinkgoRecover()
					mon.gracePeriod = 1
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				close(stopChan)
				noExitCheck := time.After(10 * time.Second)
				exited := false

				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(BeTrue())
			})
		})
	})
})
