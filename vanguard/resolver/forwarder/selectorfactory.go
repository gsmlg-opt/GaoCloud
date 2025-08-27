package forwarder

type FwdSelectPolicy int

const (
	fixedOrder FwdSelectPolicy = 0
	rttBased   FwdSelectPolicy = 1
	roundRobin FwdSelectPolicy = 2
)

func CreateSelector(policy FwdSelectPolicy, fwders []SafeFwder) FwderSelector {
	var selector FwderSelector
	switch policy {
	case fixedOrder:
		selector = newFixOrderSelector(fwders)
	case rttBased:
		selector = newRttBasedSelector(fwders)
	case roundRobin:
		selector = newRoundRobinSelector(fwders)
	default:
		panic("unknown selector policy")
	}
	return selector
}
