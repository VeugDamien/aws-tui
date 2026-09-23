package awsconfig

import "sort"

// GroupKind identifies why a set of profiles is grouped together.
type GroupKind int

const (
	// GroupSSO groups every profile sharing the same sso_session.
	GroupSSO GroupKind = iota
	// GroupDependency groups a parent profile with the profiles that derive from
	// it (via source_profile or a credential_process reference).
	GroupDependency
	// GroupStandalone holds profiles that belong to no other group.
	GroupStandalone
)

// GroupedProfile is a profile positioned within a group, carrying its display
// depth (0 for a group's top-level entries, 1+ for dependent children).
type GroupedProfile struct {
	Profile Profile
	Depth   int
}

// Group is a labelled collection of related profiles rendered together.
type Group struct {
	Kind GroupKind
	// Label is the group header (e.g. the sso-session name or parent profile).
	Label string
	// Profiles are the ordered members, already flattened with their depth so a
	// dependency chain reads parent-first, children indented underneath.
	Profiles []GroupedProfile
}

// GroupProfiles organises a flat profile list into display groups:
//   - one group per shared sso_session,
//   - one group per parent profile that has dependent children,
//   - a final "standalone" group for everything else.
//
// A profile appears in exactly one group. Precedence when a profile could fit
// several groups: it stays with its parent's dependency group if that parent is
// itself standalone; SSO membership wins when the profile carries an sso_session.
func GroupProfiles(profiles []Profile) []Group {
	byName := make(map[string]Profile, len(profiles))
	for _, p := range profiles {
		byName[p.Name] = p
	}

	// children maps a parent name to the profiles that depend on it (only when
	// the parent actually exists in the config).
	children := make(map[string][]Profile)
	for _, p := range profiles {
		parent := p.SourceProfile
		if parent == "" {
			continue
		}
		if _, ok := byName[parent]; ok {
			children[parent] = append(children[parent], p)
		}
	}

	assigned := make(map[string]bool)

	var groups []Group

	// 1) SSO session groups, ordered by session name for stability.
	ssoMembers := make(map[string][]Profile)
	var ssoNames []string
	for _, p := range profiles {
		if p.AuthKind == AuthSSO && p.SSOSession != "" {
			if _, seen := ssoMembers[p.SSOSession]; !seen {
				ssoNames = append(ssoNames, p.SSOSession)
			}
			ssoMembers[p.SSOSession] = append(ssoMembers[p.SSOSession], p)
		}
	}
	sort.Strings(ssoNames)
	for _, session := range ssoNames {
		members := ssoMembers[session]
		sortProfiles(members)
		g := Group{Kind: GroupSSO, Label: session}
		for _, p := range members {
			assigned[p.Name] = true
			g.Profiles = append(g.Profiles, GroupedProfile{Profile: p, Depth: 0})
			// Attach dependents of an SSO profile directly beneath it.
			appendChildren(&g, p, children, assigned, 1)
		}
		groups = append(groups, g)
	}

	// 2) Dependency groups: a standalone parent (not already in an SSO group)
	// that has at least one child. Ordered by parent name.
	var parents []string
	for parent := range children {
		parents = append(parents, parent)
	}
	sort.Strings(parents)
	for _, parent := range parents {
		if assigned[parent] {
			continue // parent already shown (e.g. inside an SSO group)
		}
		p := byName[parent]
		g := Group{Kind: GroupDependency, Label: parent}
		assigned[parent] = true
		g.Profiles = append(g.Profiles, GroupedProfile{Profile: p, Depth: 0})
		appendChildren(&g, p, children, assigned, 1)
		groups = append(groups, g)
	}

	// 3) Standalone group: everything not yet assigned.
	var standalone []Profile
	for _, p := range profiles {
		if !assigned[p.Name] {
			standalone = append(standalone, p)
		}
	}
	if len(standalone) > 0 {
		sortProfiles(standalone)
		g := Group{Kind: GroupStandalone, Label: ""}
		for _, p := range standalone {
			assigned[p.Name] = true
			g.Profiles = append(g.Profiles, GroupedProfile{Profile: p, Depth: 0})
		}
		groups = append(groups, g)
	}

	return groups
}

// appendChildren recursively appends the dependents of parent (deepest chains
// supported), marking each as assigned to avoid duplicates and cycles.
func appendChildren(g *Group, parent Profile, children map[string][]Profile, assigned map[string]bool, depth int) {
	kids := children[parent.Name]
	sortProfiles(kids)
	for _, c := range kids {
		if assigned[c.Name] {
			continue
		}
		assigned[c.Name] = true
		g.Profiles = append(g.Profiles, GroupedProfile{Profile: c, Depth: depth})
		appendChildren(g, c, children, assigned, depth+1)
	}
}

func sortProfiles(ps []Profile) {
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
}
