package mutating

import (
	"context"
	altershield "github.com/traas-stack/altershield-operator/pkg/altershield/client"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	updateHandlers = make(map[metav1.GroupVersionKind]handlerFunc)
)

type handlerFunc func(ctx context.Context, req *admission.Request, handler *Handler) admission.Response

type Handler struct {
	Decoder *admission.Decoder
	Cache   cache.Cache
	Client  client.Client

	altershieldClient *altershield.AltershieldClient
}

func NewHandler(altershieldEndpoint string) *Handler {
	return &Handler{
		altershieldClient: altershield.NewAltershieldClient(altershieldEndpoint),
	}
}

func (h *Handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	klog.Infof("mutating webhook request: %v", req.String())

	switch req.Operation {
	case admissionv1.Update:
		if handling, exist := updateHandlers[req.Kind]; exist {
			return handling(ctx, &req, h)
		}
	}

	return admission.Allowed("by pass")
}

func (h *Handler) InjectDecoder(decoder *admission.Decoder) error {
	h.Decoder = decoder
	return nil
}

func (h *Handler) InjectClient(client client.Client) error {
	h.Client = client
	return nil
}

func (h *Handler) InjectCache(cache cache.Cache) error {
	h.Cache = cache
	return nil
}

func gvkConverter(in schema.GroupVersionKind) metav1.GroupVersionKind {
	return metav1.GroupVersionKind{
		Group:   in.Group,
		Version: in.Version,
		Kind:    in.Kind,
	}
}
