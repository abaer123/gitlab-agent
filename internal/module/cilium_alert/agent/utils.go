package agent

import (
	"strconv"
	"strings"

	"github.com/cilium/cilium/api/v1/flow"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/policy/api"
)

const (
	alertAnnotationKey   = "app.gitlab.com/alert"
	alertAnnotationValue = "true"
)

func checkEndpointL3(cnp v2.CiliumNetworkPolicy, lbs []string) (bool, error) {
	rules, err := cnp.Parse()
	if err != nil {
		return false, err
	}
	for _, rule := range rules {
		if !existsLabelsInEndpointSelector(lbs, rule.EndpointSelector.LabelSelectorString()) {
			return false, nil
		}
	}
	return true, nil
}

func checkSourceL3(cnp v2.CiliumNetworkPolicy, lbs []string) (bool, error) {
	rls, err := cnp.Parse()
	if err != nil {
		return false, err
	}
	for _, rule := range rls {
		for _, igrs := range rule.Ingress {
			for _, eps := range igrs.FromEndpoints {
				if !existsLabelsInEndpointSelector(lbs, eps.LabelSelectorString()) {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func checkDestinationL3(cnp v2.CiliumNetworkPolicy, lbs []string) (bool, error) {
	rls, err := cnp.Parse()
	if err != nil {
		return false, err
	}
	for _, rule := range rls {
		for _, egrs := range rule.Egress {
			for _, eps := range egrs.ToEndpoints {
				if !existsLabelsInEndpointSelector(lbs, eps.LabelSelectorString()) {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func existsLabelsInEndpointSelector(lbs []string, sel string) bool {
	for _, lsel := range strings.Split(sel, ",") {
		if strings.Contains(lsel, "io.kubernetes.pod.namespace") {
			continue
		}
		lsel = strings.TrimSpace(strings.Replace(lsel, "any.", "", -1))
		exists := false
		for _, lbl := range lbs {
			if strings.Contains(lbl, lsel) {
				exists = true
				break
			}
		}
		if !exists {
			return false
		}
	}
	return true
}

func matchL4Info(flw *flow.Flow, cnp v2.CiliumNetworkPolicy) bool {
	var (
		flowPort     string
		flowProtocol api.L4Proto
	)
	l4 := flw.GetL4()
	switch l4.GetProtocol().(type) {
	case *flow.Layer4_TCP:
		flowPort = strconv.FormatUint(uint64(l4.GetTCP().GetDestinationPort()), 10)
		flowProtocol = api.ProtoTCP
	case *flow.Layer4_UDP:
		flowPort = strconv.FormatUint(uint64(l4.GetUDP().GetDestinationPort()), 10)
		flowProtocol = api.ProtoUDP
	}
	rls, err := cnp.Parse()
	if err != nil {
		return false
	}
	for _, rule := range rls {
		switch flw.GetTrafficDirection() { // nolint: exhaustive
		case flow.TrafficDirection_INGRESS:
			for _, igrs := range rule.Ingress {
				for _, tPt := range igrs.ToPorts {
					for _, pt := range tPt.Ports {
						if pt.Protocol == flowProtocol && pt.Port == flowPort {
							return true
						}
					}
				}
			}
		case flow.TrafficDirection_EGRESS:
			for _, egrs := range rule.Egress {
				for _, tPt := range egrs.ToPorts {
					for _, pt := range tPt.Ports {
						if pt.Protocol == flowProtocol && pt.Port == flowPort {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func getPolicy(flw *flow.Flow, cnps *v2.CiliumNetworkPolicyList) (*v2.CiliumNetworkPolicy, error) {
	for _, cnp := range cnps.Items {
		if cnp.Annotations[alertAnnotationKey] != alertAnnotationValue {
			continue
		}
		var (
			edp    bool
			srcdst bool
			err    error
		)
		switch flw.GetTrafficDirection() { // nolint: exhaustive
		case flow.TrafficDirection_INGRESS:
			edp, err = checkEndpointL3(cnp, flw.GetDestination().GetLabels())
			if err != nil {
				return nil, err
			}
			srcdst, err = checkSourceL3(cnp, flw.GetSource().GetLabels())
			if err != nil {
				return nil, err
			}
		case flow.TrafficDirection_EGRESS:
			edp, err = checkEndpointL3(cnp, flw.GetSource().GetLabels())
			if err != nil {
				return nil, err
			}
			srcdst, err = checkDestinationL3(cnp, flw.GetDestination().GetLabels())
			if err != nil {
				return nil, err
			}
		}
		if edp && (!srcdst || !matchL4Info(flw, cnp)) {
			return &cnp, nil
		}
	}
	return nil, nil
}

func getNamespace(flw *flow.Flow) string {
	switch flw.GetTrafficDirection() { // nolint: exhaustive
	case flow.TrafficDirection_INGRESS:
		return flw.GetDestination().GetNamespace()
	case flow.TrafficDirection_EGRESS:
		return flw.GetSource().GetNamespace()
	}
	return ""
}
