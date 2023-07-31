package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kubevirt/kubevirt-tekton-tasks/modules/tests/test/constants"
	framework2 "github.com/kubevirt/kubevirt-tekton-tasks/modules/tests/test/framework"
	"github.com/kubevirt/kubevirt-tekton-tasks/modules/tests/test/tekton"
	"github.com/kubevirt/kubevirt-tekton-tasks/modules/tests/test/testconfigs"
	. "github.com/onsi/gomega"
	pipev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tkntest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

type TaskRunRunner struct {
	framework *framework2.Framework
	taskRun   *pipev1beta1.TaskRun
	logs      string
	pod       *corev1.Pod
	err       error
}

func NewTaskRunRunner(framework *framework2.Framework, taskRun *pipev1beta1.TaskRun) *TaskRunRunner {
	Expect(taskRun).ShouldNot(BeNil())
	return &TaskRunRunner{
		framework: framework,
		taskRun:   taskRun,
	}
}

func (r *TaskRunRunner) PrintVMInfo(namespace, name string) {
	vm, err := r.framework.KubevirtClient.VirtualMachine(namespace).Get(name, &metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	vmBytes, err := json.Marshal(vm)
	Expect(err).ShouldNot(HaveOccurred())

	fmt.Println("VM definition")
	fmt.Println(string(vmBytes))
	fmt.Println("----------------------------------------------")
}

func (r *TaskRunRunner) PrintDatavolumeInfo(namespace, name string) {
	dv, err := r.framework.CdiClient.DataVolumes(namespace).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	dvBytes, err := json.Marshal(dv)
	Expect(err).ShouldNot(HaveOccurred())

	fmt.Println("Datavolume definition")
	fmt.Println(string(dvBytes))
	fmt.Println("----------------------------------------------")
	pvc, err := r.framework.CoreV1Client.PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	pvcBytes, err := json.Marshal(pvc)
	Expect(err).ShouldNot(HaveOccurred())
	fmt.Println("PVC definition")
	fmt.Println(string(pvcBytes))
	fmt.Println("----------------------------------------------")
}

func (r *TaskRunRunner) PrintTektonInfo() {
	taskBytes, err := json.Marshal(r.GetTaskRun())
	Expect(err).ShouldNot(HaveOccurred())

	podBytes, err := json.Marshal(r.GetTaskRunPod())
	Expect(err).ShouldNot(HaveOccurred())

	fmt.Println("Task definition")
	fmt.Println(string(taskBytes))
	fmt.Println("----------------------------------------------")
	fmt.Println("Pod definition")
	fmt.Println(string(podBytes))
	fmt.Println("----------------------------------------------")

	if r.logs != "" {
		fmt.Println("Pod logs")
		fmt.Println(r.logs)
		fmt.Println("----------------------------------------------")
	}
}

func (r *TaskRunRunner) GetTaskRun() *pipev1beta1.TaskRun {
	return r.taskRun
}

func (r *TaskRunRunner) GetTaskRunPod() *corev1.Pod {
	return r.pod
}

func (r *TaskRunRunner) GetError() error {
	return r.err
}

func (r *TaskRunRunner) CreateTaskRun() *TaskRunRunner {
	taskRun, err := r.framework.TknClient.TaskRuns(r.taskRun.Namespace).Create(context.TODO(), r.taskRun, v1.CreateOptions{})
	Expect(err).ShouldNot(HaveOccurred())
	r.taskRun = taskRun
	r.framework.ManageTaskRuns(r.taskRun)
	return r
}

func (r *TaskRunRunner) ExpectFailure() *TaskRunRunner {
	r.taskRun, r.logs, r.pod, r.err = tekton.WaitForTaskRunState(r.framework.Clients, r.taskRun.Namespace, r.taskRun.Name,
		r.taskRun.GetTimeout(context.TODO())+constants.Timeouts.TaskRunExtraWaitDelay.Duration,
		tkntest.TaskRunFailed(r.taskRun.Name))
	return r
}

func (r *TaskRunRunner) WaitForTaskRunFinish() *TaskRunRunner {
	r.taskRun, r.logs, r.pod, r.err = tekton.WaitForTaskRunState(r.framework.Clients, r.taskRun.Namespace, r.taskRun.Name,
		r.taskRun.GetTimeout(context.TODO())+constants.Timeouts.TaskRunExtraWaitDelay.Duration,
		func(accessor apis.ConditionAccessor) (bool, error) {
			succeeded, _ := tkntest.TaskRunSucceed(r.taskRun.Name)(accessor)
			return succeeded, nil
		})
	return r
}

func (r *TaskRunRunner) ExpectSuccess() *TaskRunRunner {
	r.taskRun, r.logs, r.pod, r.err = tekton.WaitForTaskRunState(r.framework.Clients, r.taskRun.Namespace, r.taskRun.Name,
		r.taskRun.GetTimeout(context.TODO())+constants.Timeouts.TaskRunExtraWaitDelay.Duration,
		tkntest.TaskRunSucceed(r.taskRun.Name))
	return r
}

func (r *TaskRunRunner) ExpectSuccessOrFailure(expectSuccess bool) *TaskRunRunner {
	if expectSuccess {
		r.ExpectSuccess()
	} else {
		r.ExpectFailure()
	}
	return r
}

func (r *TaskRunRunner) ExpectLogs(logs ...string) *TaskRunRunner {
	if len(logs) > 0 {
		for _, snippet := range logs {
			Expect(r.logs).Should(ContainSubstring(snippet))
		}
	}
	return r
}

func (r *TaskRunRunner) ExpectTermination(termination *testconfigs.TaskRunExpectedTermination) *TaskRunRunner {
	if termination != nil {
		Expect(r.taskRun.Status.Steps[0].Terminated.ExitCode).Should(Equal(termination.ExitCode))
	}

	return r
}

func (r *TaskRunRunner) GetResults() map[string]string {
	return tekton.TaskResultsToMap(r.taskRun.Status.TaskRunResults)
}

func (r *TaskRunRunner) ExpectResults(results map[string]string) *TaskRunRunner {
	return r.ExpectResultsWithLen(results, len(results))
}

func (r *TaskRunRunner) ExpectResultsWithLen(results map[string]string, expectedLen int) *TaskRunRunner {
	receivedResults := r.GetResults()

	Expect(receivedResults).Should(HaveLen(expectedLen))

	for resultKey, resultValue := range results {
		Expect(receivedResults[resultKey]).To(Equal(resultValue))
	}
	return r
}
