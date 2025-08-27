package k8s

import (
	"errors"
	"net"
	"strings"

	"cement/domaintree"
	"g53"
	corev1 "k8s.io/api/core/v1"
	"vanguard/config"
	"vanguard/core"
	"vanguard/resolver/auth"
	"vanguard/resolver/auth/zone"
	"vanguard/resolver/auth/zone/memoryzone"
)

var (
	errUnsupportNetmask = errors.New("only support 8, 16, 24 bits network mask")
	errInvalidClusterIP = errors.New("cluster ip isn't valid")
)

const (
	defaultTTL       = g53.RRTTL(5)
	defaultWeight    = 100
	defaultPriority  = 10
	reverseBaseZone  = "in-addr.arpa"
	versionQuery     = "dns-version"
	dnsSchemaVersion = "1.0.1"
)

type Auth struct {
	serviceZone        zone.Zone
	serviceReverseZone zone.Zone
	podReverseZone     zone.Zone
}

func NewAuth(conf *config.VanguardConf) (*Auth, error) {
	a := &Auth{}
	if err := a.reloadConfig(conf); err != nil {
		return nil, err
	} else {
		return a, nil
	}
}

func (a *Auth) reloadConfig(conf *config.VanguardConf) error {
	if err := a.createServiceZone(&conf.Kubernetes); err != nil {
		return err
	}
	return a.createReverseZone(&conf.Kubernetes)
}

func (a *Auth) createServiceZone(k8sCfg *config.Kubernetes) error {
	z, err := a.createZone(k8sCfg.ClusterDomain, k8sCfg)
	if err == nil {
		a.serviceZone = z

		n, _ := g53.NameFromStringUnsafe(versionQuery).Concat(z.GetOrigin())
		rdata, _ := g53.TxtFromString(dnsSchemaVersion)
		a.replaceRRset(z, n, g53.RR_TXT, &g53.RRset{
			Name:   n,
			Type:   g53.RR_TXT,
			Ttl:    defaultTTL,
			Class:  g53.CLASS_IN,
			Rdatas: []g53.Rdata{rdata},
		})
	}

	return err
}

func (a *Auth) createReverseZone(k8sCfg *config.Kubernetes) error {
	serviceReverseName, err := reverseZoneNameForNetwork(k8sCfg.ClusterServiceIPRange)
	if err != nil {
		return err
	}

	if z, err := a.createZone(serviceReverseName, k8sCfg); err != nil {
		return err
	} else {
		a.serviceReverseZone = z
	}

	podNetworkReverseName, err := reverseZoneNameForNetwork(k8sCfg.ClusterCIDR)
	if err != nil {
		return err
	}

	z, err := a.createZone(podNetworkReverseName, k8sCfg)
	if err == nil {
		a.podReverseZone = z
	}
	return err
}

func (a *Auth) createZone(origin string, k8sCfg *config.Kubernetes) (zone.Zone, error) {
	n, err := g53.NameFromString(origin)
	if err != nil {
		return nil, err
	}

	soaRdata := &g53.SOA{
		MName:   g53.NameFromStringUnsafe("ns.dns." + origin),
		RName:   g53.NameFromStringUnsafe("hostmaster." + origin),
		Serial:  1981616,
		Refresh: 7200,
		Retry:   1800,
		Expire:  86400,
		Minimum: 30,
	}
	soa := &g53.RRset{
		Name:   n,
		Type:   g53.RR_SOA,
		Class:  g53.CLASS_IN,
		Ttl:    defaultTTL,
		Rdatas: []g53.Rdata{soaRdata},
	}

	nsName := g53.NameFromStringUnsafe("ns.dns." + origin)
	nsRdata := &g53.NS{
		Name: nsName,
	}
	ns := &g53.RRset{
		Name:   n,
		Type:   g53.RR_NS,
		Class:  g53.CLASS_IN,
		Ttl:    defaultTTL,
		Rdatas: []g53.Rdata{nsRdata},
	}

	aRdata, err := g53.AFromString(k8sCfg.ClusterDNSServer)
	if err != nil {
		return nil, err
	}

	glue := &g53.RRset{
		Name:   nsName,
		Type:   g53.RR_A,
		Class:  g53.CLASS_IN,
		Ttl:    defaultTTL,
		Rdatas: []g53.Rdata{aRdata},
	}

	z := memoryzone.NewDynamicZone(n)
	up, _ := z.GetUpdator(nil, true)
	tx, _ := up.Begin()
	up.Add(tx, soa)
	up.Add(tx, ns)
	up.Add(tx, glue)
	tx.Commit()
	return z, nil
}

func (a *Auth) resolve(client *core.Client) bool {
	request := client.Request
	zones := []zone.Zone{a.serviceZone, a.serviceReverseZone, a.podReverseZone}
	for _, z := range zones {
		if request.Question.Name.IsSubDomain(z.GetOrigin()) {
			query := auth.NewQuery(domaintree.ClosestEncloser, request, z)
			query.Process()
			client.Response = query.GetResponse()
			client.CacheAnswer = false
			return true
		}
	}
	return false
}

func reverseZoneNameForNetwork(network string) (string, error) {
	_, net, err := net.ParseCIDR(network)
	if err != nil {
		return "", err
	}

	ipLabels := strings.Split(network, ".")
	ones, _ := net.Mask.Size()
	switch ones {
	case 8:
		return strings.Join([]string{ipLabels[0], reverseBaseZone}, "."), nil
	case 16:
		return strings.Join([]string{ipLabels[1], ipLabels[0], reverseBaseZone}, "."), nil
	case 24:
		return strings.Join([]string{ipLabels[2], ipLabels[1], ipLabels[0], reverseBaseZone}, "."), nil
	default:
		return "", errUnsupportNetmask
	}
}

func (a *Auth) replaceServiceRRset(name *g53.Name, typ g53.RRType, rrset *g53.RRset) error {
	return a.replaceRRset(a.serviceZone, name, typ, rrset)
}

func (a *Auth) replacePodReverseRRset(name *g53.Name, typ g53.RRType, rrset *g53.RRset) error {
	return a.replaceRRset(a.podReverseZone, name, typ, rrset)
}

func (a *Auth) replaceServiceReverseRRset(name *g53.Name, typ g53.RRType, rrset *g53.RRset) error {
	return a.replaceRRset(a.serviceReverseZone, name, typ, rrset)
}

func (a *Auth) replaceRRset(z zone.Zone, name *g53.Name, typ g53.RRType, rrset *g53.RRset) error {
	up, _ := z.GetUpdator(nil, true)
	tx, _ := up.Begin()
	if rrset != nil {
		up.DeleteRRset(tx, rrset)
		up.Add(tx, rrset)
	} else {
		up.DeleteRRset(tx, &g53.RRset{
			Name: name,
			Type: typ,
		})
	}
	tx.Commit()
	return nil
}

func (a *Auth) getServiceDomain(svc *corev1.Service) *g53.Name {
	n, _ := g53.NameFromStringUnsafe(strings.Join([]string{svc.Name, svc.Namespace, "svc"}, ".")).Concat(a.serviceZone.GetOrigin())
	return n
}

func (a *Auth) getEndpointsAddrDomain(addr *corev1.EndpointAddress, svc, namespace string) *g53.Name {
	podName := addr.Hostname
	if podName == "" {
		podName = strings.Replace(addr.IP, ".", "-", 3)
	}
	n, _ := g53.NameFromStringUnsafe(strings.Join([]string{podName, svc, namespace, "svc"}, ".")).Concat(a.serviceZone.GetOrigin())
	return n
}

func (a *Auth) getPortName(port, protocol, svc, namespace string) *g53.Name {
	n, _ := g53.NameFromStringUnsafe(strings.Join([]string{"_" + port, "_" + protocol, svc, namespace, "svc"}, ".")).Concat(a.serviceZone.GetOrigin())
	return n
}

func (a *Auth) getReverseName(ip string) (*g53.Name, error) {
	labels := strings.Split(ip, ".")
	if len(labels) != 4 {
		return nil, errInvalidClusterIP
	} else {
		return g53.NameFromStringUnsafe(strings.Join([]string{labels[3], labels[2], labels[1], labels[0], reverseBaseZone}, ".")), nil
	}
}
