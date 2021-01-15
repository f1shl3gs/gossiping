package cluster

import (
	"github.com/hashicorp/memberlist"
)

type Action string

const (
	PeerJoin   Action = "join"
	PeerUpdate        = "update"
	PeerLeave         = "leave"
)

type PeerObservation struct {
	Action Action
	Node   *memberlist.Node
}
