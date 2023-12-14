package validating

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	deleteHandlers = make(map[metav1.GroupVersionKind]handlerFunc)
)

type handlerFunc func(ctx context.Context, req *admission.Request, handler *Handler) admission.Response

type Handler struct {
	Decoder *admission.Decoder
	Cache   cache.Cache
	Client  client.Client
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case admissionv1.Delete:
		if handling, exist := deleteHandlers[req.Kind]; exist {
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
