package authz_test

import (
	"fmt"
	"github.com/coder/coder/coderd/authz/authztest"
	"math/bits"
	"testing"
)

var nilSet = authztest.Set{nil}

func Test_ExhaustiveAuthorize(t *testing.T) {
	all := authztest.GroupedPermissions(authztest.AllPermissions())
	variants := permissionVariants(all)
	var total int
	for name, v := range variants {
		fmt.Printf("%s: %d\n", name, v.Size())
		total += v.Size()
	}
	fmt.Println(total)
}

func permissionVariants(all authztest.SetGroup) map[string]*authztest.Role {
	// Cases are X+/- where X indicates the level where the impactful set is.
	// The impactful set determines the result.
	variants := make(map[string]*authztest.Role)
	assignVariants(variants, "W", authztest.LevelWildKey, all)
	assignVariants(variants, "S", authztest.LevelSiteKey, all)
	assignVariants(variants, "O", authztest.LevelOrgKey, all)
	assignVariants(variants, "M", authztest.LevelOrgMemKey, all)
	assignVariants(variants, "U", authztest.LevelUserKey, all)
	return variants
}

func assignVariants(m map[string]*authztest.Role, name string, lvl authztest.LevelKey, all authztest.SetGroup) {
	vs := levelVariants(lvl, all)
	m[name+"+"] = vs[0]
	m[name+"-"] = vs[1]
}

func levelVariants(lvl authztest.LevelKey, all authztest.SetGroup) []*authztest.Role {
	ordered := []authztest.LevelKey{
		authztest.LevelWildKey,
		authztest.LevelSiteKey,
		// TODO: @emyrk orgs are special where the noise flags have to change
		//		since these two levels are the same. The current code does
		//		not handle this correctly.
		authztest.LevelOrgKey,
		authztest.LevelOrgMemKey,
		authztest.LevelUserKey,
	}

	noiseFlag := abstain
	sets := make([]authztest.Iterable, 0)
	for _, l := range ordered {
		if l == lvl {
			noiseFlag = positive | negative | abstain
			continue
		}
		sets = append(sets, noise(noiseFlag, all.Level(l)))
	}

	// clone the sets so we can get 2 sets. One for positive, one for negative
	clone := make([]authztest.Iterable, len(sets))
	copy(clone, sets)
	p := append(clone, pos(all.Level(lvl)))
	n := append(sets, neg(all.Level(lvl)))

	return []*authztest.Role{
		authztest.NewRole(p...),
		authztest.NewRole(n...),
	}
}

// pos returns the positive impactful variant for a given level. It does not
// include noise at any other level but the one given.
func pos(lvl authztest.LevelGroup) *authztest.Role {
	return authztest.NewRole(
		lvl.Positive(),
		authztest.Union(lvl.Abstain()[:1], nilSet),
	)
}

func neg(lvl authztest.LevelGroup) *authztest.Role {
	return authztest.NewRole(
		lvl.Negative(),
		authztest.Union(lvl.Positive()[:1], nilSet),
		authztest.Union(lvl.Abstain()[:1], nilSet),
	)
}

type noiseBits uint8

const (
	none noiseBits = 1 << iota
	positive
	negative
	abstain
)

func flagMatch(flag, in noiseBits) bool {
	return flag&in != 0
}

// noise returns the noise permission permutations for a given level. You can
// use this helper function when this level is not impactful.
// The returned role is the permutations including at least one example of
// positive, negative, and neutral permissions. It also includes the set of
// no additional permissions.
func noise(f noiseBits, lvls ...authztest.LevelGroup) *authztest.Role {
	rs := make([]authztest.Iterable, 0, len(lvls))
	for _, lvl := range lvls {
		sets := make([]authztest.Iterable, 0, bits.OnesCount8(uint8(f)))

		if flagMatch(positive, f) {
			sets = append(sets, authztest.Union(lvl.Positive()[:1], nilSet))
		}
		if flagMatch(negative, f) {
			sets = append(sets, authztest.Union(lvl.Negative()[:1], nilSet))
		}
		if flagMatch(abstain, f) {
			sets = append(sets, authztest.Union(lvl.Abstain()[:1], nilSet))
		}

		rs = append(rs, authztest.NewRole(
			sets...,
		))
	}

	if len(rs) == 1 {
		return rs[0].(*authztest.Role)
	}
	return authztest.NewRole(rs...)
}
