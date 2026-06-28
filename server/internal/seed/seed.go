// Package seed ports the original in-app demo data into PostgreSQL. It is
// idempotent: if any swimlane already exists, it does nothing.
package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/tpasson/sw-atlas/server/internal/store"
)

// Run seeds demo data into the given workspace and returns the resulting
// swimlane count.
func Run(ctx context.Context, st *store.Store, ws string) (int, error) {
	if n, err := st.CountSwimlanes(ctx, ws); err != nil || n > 0 {
		return n, err
	}

	year := time.Now().Year()
	d := func(month, day int) *string {
		s := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		return &s
	}

	type lane struct {
		id, name, color string
		subs            [][2]string // {id, name}
	}
	lanes := []lane{
		{"ca", "Customer Accounts", "#0A84FF", [][2]string{
			{"ca-con", "Contoso"}, {"ca-nor", "Northwind"}, {"ca-glo", "Globex"},
			{"ca-acm", "Acme"}, {"ca-fab", "Fabrikam"}, {"ca-ini", "Initech"},
		}},
		{"ev", "Events", "#FFCC00", [][2]string{
			{"ev-conf", "Annual Conference"}, {"ev-summit", "Industry Summit"}, {"ev-launch", "Product Launch"},
		}},
		{"pr", "Product", "#30D158", [][2]string{
			{"pr-disc", "Discovery"}, {"pr-des", "Design"},
			{"pr-proto", "Prototyping"}, {"pr-beta", "Beta Program"},
		}},
		{"pf", "Platform", "#FF9F0A", [][2]string{
			{"pf-api", "API"}, {"pf-web", "Web App"}, {"pf-mob", "Mobile"},
			{"pf-infra", "Infrastructure"}, {"pf-data", "Data & Analytics"},
		}},
		{"fn", "Features", "#FF375F", [][2]string{
			{"fn-fd", "Feature Development"}, {"fn-qa", "QA & Validation"},
			{"fn-int", "Integration"}, {"fn-comp", "Compliance"},
		}},
		{"tl", "Tools", "#BF5AF2", [][2]string{
			{"tl-a", "Tool Alpha"}, {"tl-b", "Tool Beta"}, {"tl-g", "Tool Gamma"},
		}},
		{"rm", "Release Management", "#32ADE6", [][2]string{
			{"rm-sp", "Sprint Planning"}, {"rm-qa", "QA & Testing"},
			{"rm-so", "Sign-off"}, {"rm-de", "Deployment"},
		}},
	}
	for _, l := range lanes {
		if _, err := st.CreateSwimlane(ctx, ws, l.id, l.name, l.color); err != nil {
			return 0, err
		}
		for _, s := range l.subs {
			if _, err := st.CreateSubLane(ctx, ws, l.id, s[0], s[1]); err != nil {
				return 0, err
			}
		}
	}

	ms := func(id, sw, sub string, month int, title, what, why, how, who string, when *string) store.Item {
		var subp *string
		if sub != "" {
			s := sub
			subp = &s
		}
		mat := ((month - 1) % 4) + 1 // demo spread across the 4 maturity stages
		prog := mat*25 - 10         // demo % progress, varies with the stage
		return store.Item{
			ID: id, SwimlaneID: sw, SubLaneID: subp, Year: year, Month: month,
			Title: title, What: what, Why: why, How: how, Who: who, When: when,
			Kind: "milestone", Marker: "l:Diamond", Maturity: &mat, Progress: &prog,
		}
	}

	items := []store.Item{
		ms("m-con-rev", "ca", "ca-con", 6, "Solution Review", "Present the platform to Contoso", "Strategic partnership potential", "Contoso HQ", "Sales Lead", d(6, 15)),
		ms("m-con-fw", "ca", "ca-con", 9, "Framework Agreement", "Sign framework agreement for next year", "Lock in Contoso as anchor customer", "Legal negotiation", "Account Exec + Legal", d(9, 15)),
		ms("m-nor-poc", "ca", "ca-nor", 5, "Proof of Concept", "PoC for the Northwind use case", "Gateway to a full rollout", "Joint workshop", "Key Account Manager", d(5, 29)),
		ms("m-glo-rev", "ca", "ca-glo", 7, "Account Review", "Annual review with Globex", "Prepare the renewal", "Business review deck", "Customer Success", d(7, 9)),
		ms("m-acm-jda", "ca", "ca-acm", 7, "Partnership Signed", "Joint development agreement", "Build a shared integration", "Contract + NDA", "Business Dev", d(7, 31)),

		ms("m-conf", "ev", "ev-conf", 1, "Annual Conference", "Company conference & keynote", "Industry visibility", "Booth + live demo", "Marketing + Eng", d(1, 8)),
		ms("m-summit", "ev", "ev-summit", 9, "Industry Summit", "Exhibition at the industry summit", "Reach new customers", "Demo + keynote", "CEO + Marketing", d(9, 10)),

		ms("m-mvp", "pr", "pr-proto", 5, "MVP Ready", "Minimum viable product complete", "Foundation for the beta", "Core build", "Product Lead", d(5, 15)),
		ms("m-proto", "pr", "pr-proto", 3, "Interactive Prototype", "Clickable end-to-end prototype", "Validate the concept early", "Rapid prototyping", "Design Engineer", d(3, 31)),
		ms("m-beta-launch", "pr", "pr-beta", 9, "Beta Launch", "Open beta to pilot customers", "Real-world feedback", "Controlled rollout", "Product Team", d(9, 3)),

		ms("m-apiv2", "pf", "pf-api", 3, "API v2", "New public API architecture", "Reduce integration friction", "Design + implementation", "Platform Architect", d(3, 31)),
		ms("m-webr2", "pf", "pf-web", 6, "Web App R2", "Redesigned web client", "Foundation to scale", "SW + QA verification", "Web Team", d(6, 30)),
		ms("m-pipeline", "pf", "pf-infra", 7, "CI/CD Pipeline", "Automated build & deploy pipeline", "Faster, safer releases", "Infrastructure automation", "DevOps Engineer", d(7, 31)),

		ms("m-search", "fn", "fn-fd", 2, "Search & Filters", "Advanced search and filtering", "Core usability", "Development sprints", "Feature Team", d(2, 20)),
		ms("m-collab", "fn", "fn-fd", 5, "Real-time Collaboration", "Multi-user concurrent editing", "Top customer requirement", "Backend + UI", "Feature Lead", d(5, 31)),
		ms("m-sso", "fn", "fn-fd", 9, "SSO Integration", "Single sign-on (SAML / OIDC)", "Required for enterprise", "Auth integration", "Security Team", d(9, 30)),
		ms("m-qaq2", "fn", "fn-qa", 6, "QA Campaign Q2", "Full regression coverage for Q2", "Quality gate for release", "Test execution", "QA Team", d(6, 30)),
		ms("m-audit", "fn", "fn-comp", 10, "Security Audit", "External security audit", "Compliance requirement", "External auditor", "Security Lead", d(10, 31)),

		ms("m-beta", "tl", "tl-b", 3, "Beta Pilot Rollout", "Pilot with 3 teams", "Real-world feedback", "Controlled rollout", "Product Owner", d(3, 31)),
		ms("m-gamma", "tl", "tl-g", 4, "Gamma Prototype", "Working Gamma prototype", "Validate the concept", "Rapid prototyping", "Gamma Team", d(4, 30)),

		ms("m-systest", "rm", "rm-qa", 7, "System Test Campaign", "Full system test for H1", "Release readiness", "Test report", "QA Team", d(7, 31)),
		ms("m-freeze", "rm", "rm-so", 9, "Feature Freeze", "No new features after this date", "Enter stabilisation", "Management decision", "PM + CTO", d(9, 30)),
		ms("m-release", "rm", "rm-de", 12, "Annual Release", "Production release", "Deliver the annual features", "Deployment + hypercare", "Release Manager", d(12, 31)),
	}

	// One time-spanning event to exercise the bar rendering coming in P1.
	summitWeek := store.Item{
		ID: "m-summit-week", SwimlaneID: "ev", SubLaneID: strptr("ev-summit"), Year: year, Month: 9,
		Title: "Summit Week", What: "On-site exhibition week", Why: "Customer meetings", How: "Conference venue",
		Who: "Whole team", Kind: "event", Marker: "bar", StartDate: d(9, 8), EndDate: d(9, 14), When: d(9, 8),
	}
	items = append(items, summitWeek)

	// Link a few milestones to source control (GitHub/GitLab) for the SCM badge demo.
	scm := map[string]string{
		"m-apiv2":    "https://github.com/acme/platform/releases/tag/v2.0.0",
		"m-mvp":      "https://dev.azure.com/acme/Platform/_git/api/pullrequest/482",
		"m-webr2":    "https://gitea.com/acme/web/src/branch/release-r2",
		"m-pipeline": "https://bitbucket.org/acme/infra/commits/9f3c1a2b8d4e",
		"m-collab":   "https://gitlab.com/acme/app/-/merge_requests/77",
	}
	for i := range items {
		if u, ok := scm[items[i].ID]; ok {
			items[i].ScmURL = &u
		}
	}

	for _, it := range items {
		if _, err := st.CreateItem(ctx, ws, it); err != nil {
			return 0, err
		}
	}

	links := [][2]string{
		{"m-con-rev", "m-apiv2"}, {"m-con-rev", "m-mvp"}, {"m-con-rev", "m-webr2"},
		{"m-con-rev", "m-search"}, {"m-con-rev", "m-collab"}, {"m-con-fw", "m-con-rev"},
		{"m-nor-poc", "m-search"}, {"m-nor-poc", "m-proto"}, {"m-nor-poc", "m-mvp"},
		{"m-glo-rev", "m-systest"}, {"m-glo-rev", "m-mvp"},
		{"m-summit", "m-beta-launch"}, {"m-summit", "m-con-rev"}, {"m-summit", "m-nor-poc"},
		{"m-freeze", "m-release"}, {"m-sso", "m-audit"},
	}
	for _, lk := range links {
		if err := st.AddLink(ctx, ws, lk[0], lk[1], ""); err != nil {
			return 0, err
		}
	}

	return st.CountSwimlanes(ctx, ws)
}

func strptr(s string) *string { return &s }
