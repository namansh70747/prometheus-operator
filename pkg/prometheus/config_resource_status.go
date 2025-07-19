package prometheus

import (
    "context"
    "time"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
    monitoringv1ac "github.com/prometheus-operator/prometheus-operator/pkg/client/applyconfiguration/monitoring/v1"
    "github.com/prometheus-operator/prometheus-operator/pkg/operator"
)

// updateServiceMonitorStatus reconciles the Status subresource of the given ServiceMonitors.
// It is a no-op when the StatusForConfigurationResources feature-gate is disabled.
func (c *Operator) updateServiceMonitorStatus(ctx context.Context, p *monitoringv1.Prometheus, smons ResourcesSelection[*monitoringv1.ServiceMonitor]) {
    if !c.configResourcesStatusEnabled {
        return
    }

    now := metav1.NewTime(time.Now().UTC())

    for _, sel := range smons {
        sm := sel.resource
        namespace := sm.Namespace
        name := sm.Name

        conditionStatus := monitoringv1.ConditionFalse
        reason := sel.reason
        message := ""
        if sel.err != nil {
            message = sel.err.Error()
        }
        if sel.err == nil {
            conditionStatus = monitoringv1.ConditionTrue
            reason = ""
            message = ""
        }

        cond := monitoringv1ac.ConfigResourceCondition().
            WithType(monitoringv1.Accepted).
            WithStatus(conditionStatus).
            WithLastTransitionTime(now).
            WithReason(reason).
            WithMessage(message).
            WithObservedGeneration(sm.Generation)

        binding := monitoringv1ac.WorkloadBinding().
            WithGroup("monitoring.coreos.com").
            WithResource("prometheuses").
            WithName(p.Name).
            WithNamespace(p.Namespace).
            WithConditions(cond)

        status := monitoringv1ac.ConfigResourceStatus().WithBindings(binding)

        applyConf := monitoringv1ac.ServiceMonitor(name, namespace).WithStatus(status)

        _, err := c.mclient.MonitoringV1().ServiceMonitors(namespace).ApplyStatus(ctx, applyConf, metav1.ApplyOptions{FieldManager: operator.PrometheusOperatorFieldManager, Force: true})
        if err != nil {
            c.logger.Error("failed to apply ServiceMonitor status", "name", name, "namespace", namespace, "err", err)
        }
    }
}