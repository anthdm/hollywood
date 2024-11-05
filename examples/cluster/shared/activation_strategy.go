package shared

import (
	"math/rand"

	"github.com/fancom/hollywood/cluster"
)

type regionBasedActivationStrategy struct {
	fallback string
}

func RegionBasedActivationStrategy(fallback string) regionBasedActivationStrategy {
	return regionBasedActivationStrategy{
		fallback: fallback,
	}
}

func (as regionBasedActivationStrategy) ActivateOnMember(details cluster.ActivationDetails) *cluster.Member {
	members := filterMembersByRegion(details.Members, details.Region)
	if len(members) > 0 {
		return members[rand.Intn(len(members))]
	}
	// if we could not find a member for the region, try to fall back.
	members = filterMembersByRegion(details.Members, as.fallback)
	if len(members) > 0 {
		return members[rand.Intn(len(members))]
	}
	return nil
}

func filterMembersByRegion(members []*cluster.Member, region string) []*cluster.Member {
	mems := make([]*cluster.Member, 0)
	for _, member := range members {
		if member.Region == region {
			mems = append(mems, member)
		}
	}
	return mems
}
