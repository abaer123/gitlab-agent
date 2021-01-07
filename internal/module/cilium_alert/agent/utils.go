package agent

import (
	"strings"

	"github.com/cilium/cilium/api/v1/flow"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
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

func getPolicy(flw *flow.Flow, cnps *v2.CiliumNetworkPolicyList) (*v2.CiliumNetworkPolicy, error) {
	for _, cnp := range cnps.Items {
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
		if edp && !srcdst {
			return &cnp, nil
		}
	}
	return nil, nil
}
