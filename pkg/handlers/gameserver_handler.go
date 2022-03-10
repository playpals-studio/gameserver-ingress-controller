package handlers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"context"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/Octops/gameserver-ingress-controller/pkg/reconcilers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/client-go/informers/core/v1"
	networkingv1 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

type GameSeverEventHandler struct {
	logger            *logrus.Entry
	client            *kubernetes.Clientset
	serviceReconciler *reconcilers.ServiceReconciler
	ingressReconciler *reconcilers.IngressReconciler
}

func NewGameSeverEventHandler(config *rest.Config, svcInformer v1.ServiceInformer, ingressInformer networkingv1.IngressInformer, recorder record.EventRecorder) *GameSeverEventHandler {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		runtime.Logger().WithError(err).Fatal("failed to create kubernetes client")
	}

	return &GameSeverEventHandler{
		logger:            runtime.Logger().WithField("role", "event_handler"),
		client:            client,
		serviceReconciler: reconcilers.NewServiceReconciler(client, svcInformer, recorder),
		ingressReconciler: reconcilers.NewIngressReconciler(client, ingressInformer, recorder),
	}
}

func (h *GameSeverEventHandler) OnAdd(obj interface{}) error {
	h.logger.WithField("event", "added").Infof("%s", obj.(*agonesv1.GameServer).Name)

	gs := gameserver.FromObject(obj)

	if err := h.Reconcile(gs); err != nil {
		h.logger.Error(err)
	}

	return nil
}

func (h *GameSeverEventHandler) OnUpdate(_ interface{}, newObj interface{}) error {
	h.logger.WithField("event", "updated").Infof("%s", newObj.(*agonesv1.GameServer).Name)

	gs := gameserver.FromObject(newObj)

	if err := h.Reconcile(gs); err != nil {
		h.logger.Error(err)
	}

	return nil
}

func (h *GameSeverEventHandler) OnDelete(obj interface{}) error {
	h.logger.WithField("event", "deleted").Infof("%s", obj.(*agonesv1.GameServer).Name)

	return nil
}

func (h *GameSeverEventHandler) Reconcile(gs *agonesv1.GameServer) error {
	if _, ok := gameserver.HasAnnotation(gs, gameserver.OctopsAnnotationIngressMode); !ok {
		h.logger.Debugf("skipping gameserver %s/%s, annotation %s not present", gs.Namespace, gs.Name, gameserver.OctopsAnnotationIngressMode)
		return nil
	}

	if gameserver.IsReady(gs) == false {
		msg := fmt.Sprintf("gameserver %s/%s not ready", gs.Namespace, gs.Name)
		h.logger.Info(msg)

		return nil
	}

	ctx := context.TODO()
	_, err := h.serviceReconciler.Reconcile(ctx, gs)
	if err != nil {
		return errors.Wrapf(err, "failed to reconcile service %s/%s", gs.Namespace, gs.Name)
	}

	_, err = h.ingressReconciler.Reconcile(ctx, gs)
	if err != nil {
		return errors.Wrapf(err, "failed to reconcile ingress %s/%s", gs.Namespace, gs.Name)
	}

	return nil
}
