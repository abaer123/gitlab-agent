package agent

import (
	"net/url"
	"regexp"
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

func checkEndpointL3(rules api.Rules, lbs []string) bool {
	for _, rule := range rules {
		if !existsLabelsInEndpointSelector(lbs, rule.EndpointSelector) {
			return false
		}
	}
	return true
}

func checkSourceL3(rules api.Rules, lbs []string) bool {
	for _, rule := range rules {
		for _, igrs := range rule.Ingress {
			for _, eps := range igrs.FromEndpoints {
				if !existsLabelsInEndpointSelector(lbs, eps) {
					return false
				}
			}
		}
	}
	return true
}

func checkDestinationL3(rules api.Rules, lbs []string) bool {
	for _, rule := range rules {
		for _, egrs := range rule.Egress {
			for _, eps := range egrs.ToEndpoints {
				if !existsLabelsInEndpointSelector(lbs, eps) {
					return false
				}
			}
		}
	}
	return true
}

func existsLabelsInEndpointSelector(lbs []string, eps api.EndpointSelector) bool {
	sel := eps.LabelSelectorString()

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

func matchL4Info(rules api.Rules, flw *flow.Flow) bool {
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
	for _, rule := range rules {
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

func matchL7Policy(rules api.Rules, flw *flow.Flow) bool {
	flwHTTP := flw.GetL7().GetHttp()
	if flwHTTP == nil {
		return false
	}
	for _, ntwRule := range rules {
		switch flw.GetTrafficDirection() { // nolint: exhaustive
		case flow.TrafficDirection_INGRESS:
			for _, igrs := range ntwRule.Ingress {
				if matchL7Rules(igrs.ToPorts, flwHTTP) {
					return true
				}
			}
		case flow.TrafficDirection_EGRESS:
			for _, egrs := range ntwRule.Egress {
				if matchL7Rules(egrs.ToPorts, flwHTTP) {
					return true
				}
			}
		}
	}
	return false
}

func matchL7Rules(rules api.PortRules, flwHTTP *flow.HTTP) bool {
	for _, prt := range rules {
		rule := prt.Rules
		if rule == nil {
			continue
		}
		for _, prtHTTP := range rule.HTTP {
			if len(prtHTTP.Headers) != 0 {
				if !includeHeader(prtHTTP.Headers, flwHTTP.GetHeaders()) {
					continue
				}
			}
			if prtHTTP.Host == "" && prtHTTP.Path == "" && prtHTTP.Method == "" {
				return true
			}
			// The check above ensures the URL is only parsed if it's actually needed below
			flwURL, err := url.Parse(flwHTTP.GetUrl())
			if err != nil {
				continue
			}
			if prtHTTP.Host != "" {
				mch, err := regexp.MatchString(prtHTTP.Host, flwURL.Host)
				if err != nil || !mch {
					continue
				}
			}
			if prtHTTP.Path != "" {
				mch, err := regexp.MatchString(prtHTTP.Path, flwURL.Path)
				if err != nil || !mch {
					continue
				}
			}
			if prtHTTP.Method != "" {
				mch, err := regexp.MatchString(prtHTTP.Method, flwHTTP.GetMethod())
				if err != nil || !mch {
					continue
				}
			}
			return true
		}
	}
	return false
}

func includeHeader(ntwHdrs []string, flwHrds []*flow.HTTPHeader) bool {
nextNtwHdr:
	for _, ntwHdr := range ntwHdrs {
		ntwHrdSplit := strings.SplitN(ntwHdr, ":", 2)
		if len(ntwHrdSplit) != 2 {
			return false
		}
		ntwHrdKey := ntwHrdSplit[0]
		ntwHrdValue := strings.TrimLeft(ntwHrdSplit[1], " ")
		for _, flwHrd := range flwHrds {
			if flwHrd.Key == ntwHrdKey && flwHrd.Value == ntwHrdValue {
				continue nextNtwHdr
			}
		}
		return false
	}
	return true
}

func l7Applicable(rules api.Rules, dir flow.TrafficDirection) bool {
	for _, rule := range rules {
		switch dir { // nolint: exhaustive
		case flow.TrafficDirection_INGRESS:
			for _, igrs := range rule.Ingress {
				for _, tPtr := range igrs.ToPorts {
					if tPtr.Rules != nil && len(tPtr.Rules.HTTP) != 0 {
						return true
					}
				}
			}
		case flow.TrafficDirection_EGRESS:
			for _, egrs := range rule.Egress {
				for _, tPtr := range egrs.ToPorts {
					if tPtr.Rules != nil && len(tPtr.Rules.HTTP) != 0 {
						return true
					}
				}
			}
		}
	}
	return false
}

func getPolicy(flw *flow.Flow, cnps []interface{}) (*v2.CiliumNetworkPolicy, error) {
	for _, cnpTmp := range cnps {
		cnp := cnpTmp.(*v2.CiliumNetworkPolicy)

		if cnp.Annotations[alertAnnotationKey] != alertAnnotationValue {
			continue
		}
		rules, err := cnp.Parse()
		if err != nil {
			return nil, err
		}
		var srcdst bool
		switch flw.GetTrafficDirection() { // nolint: exhaustive
		case flow.TrafficDirection_INGRESS:
			if !checkEndpointL3(rules, flw.GetDestination().GetLabels()) {
				continue
			}
			srcdst = checkSourceL3(rules, flw.GetSource().GetLabels())
		case flow.TrafficDirection_EGRESS:
			if !checkEndpointL3(rules, flw.GetSource().GetLabels()) {
				continue
			}
			srcdst = checkDestinationL3(rules, flw.GetDestination().GetLabels())
		default: // TrafficDirection_TRAFFIC_DIRECTION_UNKNOWN or something else
			continue
		}
		if !srcdst || !matchL4Info(rules, flw) || (l7Applicable(rules, flw.GetTrafficDirection()) && !matchL7Policy(rules, flw)) {
			return cnp, nil
		}
	}
	return nil, nil
}
