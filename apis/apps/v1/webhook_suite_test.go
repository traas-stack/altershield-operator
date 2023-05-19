/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	//+kubebuilder:scaffold:imports
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "config", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	// err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&DeploymentWebhook{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Deployment Validator", func() {
	var decoder *admission.Decoder
	var validator *DeploymentValidator

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		err := v1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		decoder, err = admission.NewDecoder(scheme)
		Expect(err).NotTo(HaveOccurred())

		validator = &DeploymentValidator{
			decoder: decoder,
		}
	})

	It("should deny an invalid Deployment", func() {
		// create an invalid Deployment
		oldDeployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Labels:    map[string]string{utils.SuspendLabel: strconv.FormatInt(time.Now().Unix(), utils.NumberTen)},
			},
			Spec: v1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
			},
		}
		newDeployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
			Spec: v1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
			},
		}

		// encode the Deployment as a raw JSON object
		deploymentJSON, err := json.Marshal(newDeployment)
		Expect(err).NotTo(HaveOccurred())
		oldDeploymentJSON, err := json.Marshal(oldDeployment)
		Expect(err).NotTo(HaveOccurred())

		// create an admission request with the raw JSON object
		admissionRequest := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:      metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
				Resource:  metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
				Operation: admissionv1.Update,
				Object:    runtime.RawExtension{Raw: deploymentJSON},
				OldObject: runtime.RawExtension{Raw: oldDeploymentJSON},
			},
		}

		// validate the request with the DeploymentValidator
		response := validator.Handle(context.Background(), admissionRequest)

		// verify that the response is a denied response
		Expect(response.Allowed).To(BeFalse())
		Expect(string(response.Result.Reason)).To(Equal("deployment test-deployment is suspended"))
	})

	It("should allow a valid Deployment", func() {
		// create a valid Deployment
		deployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
			Spec: v1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "test"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		// encode the Deployment as a raw JSON object
		deploymentJSON, err := json.Marshal(deployment)
		Expect(err).NotTo(HaveOccurred())

		// create an admission request with the raw JSON object
		admissionRequest := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Kind:      metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
				Resource:  metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
				Operation: admissionv1.Create,
				Object:    runtime.RawExtension{Raw: deploymentJSON},
			},
		}

		// validate the request with the DeploymentValidator
		response := validator.Handle(context.Background(), admissionRequest)

		// verify that the response is an allowed response
		Expect(response.Allowed).To(BeTrue())
	})
})
