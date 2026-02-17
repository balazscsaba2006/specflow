package hardq

// EntityType identifies the kind of entity for question lookup.
type EntityType string

const (
	Initiative EntityType = "initiative"
	Epic       EntityType = "epic"
	Story      EntityType = "story"
	PRD        EntityType = "prd"
	TechSpec   EntityType = "tech-spec"
	APISpec    EntityType = "api-spec"
	DesignSpec EntityType = "design-spec"
	ADR        EntityType = "adr"
	OnePager   EntityType = "one-pager"
)

// Questions returns the hard questions for the given entity type.
// Returns nil if no questions are defined for the type.
func Questions(t EntityType) []string {
	return questions[t]
}

var questions = map[EntityType][]string{
	Initiative: {
		"What's the minimum viable version of this initiative?",
		"What happens if this takes 3x longer than planned?",
		"Which epics are P0 (must-have) vs P1 (nice-to-have)?",
		"What's the failure mode if we ship 60% of this?",
		"What external dependencies could block this?",
		"What do we need to learn before we can plan this properly?",
		"How will we know this initiative is \"done\"?",
	},
	Epic: {
		"What's the boundary of this epic? Where exactly does it end?",
		"What happens at 10x scale?",
		"What's the migration path from current state to this?",
		"What coupling does this introduce between modules/systems?",
		"Is this over-engineered for now or under-engineered for later?",
		"What's the rollback plan if this goes wrong?",
		"Can this be shipped incrementally, or is it all-or-nothing?",
		"What's the simplest version that validates the core hypothesis?",
	},
	Story: {
		"Is this actually one story or multiple?",
		"What edge cases aren't covered by the acceptance criteria?",
		"What assumptions are baked into this story?",
		"What will break if requirements change after this ships?",
		"Is this testable without manual verification?",
		"What's the blast radius if this implementation is wrong?",
	},
	PRD: {
		"Is the problem validated with evidence or assumed?",
		"Are the success metrics actually measurable with current tooling?",
		"What's explicitly out of scope, and is that the right boundary?",
		"Who loses if we build this? What's the second-order effect?",
		"What's the simplest version that validates the hypothesis?",
		"Are there regulatory or compliance implications not mentioned?",
	},
	TechSpec: {
		"What happens at 10x the expected load?",
		"What's the failure mode? What breaks first?",
		"What's the migration path from current state?",
		"Are there coupling decisions that will be expensive to reverse?",
		"Is the testing strategy sufficient? What's untested?",
		"What's the operational overhead of this approach?",
	},
}

// ReviewPrompt returns the review prompt template for the given doc type.
// The template contains a {{doc_content}} placeholder.
// Returns empty string if no review prompt is defined for the type.
func ReviewPrompt(t EntityType) string {
	return reviewPrompts[t]
}

var reviewPrompts = map[EntityType]string{
	PRD: `Review the following PRD as a senior product architect.

Focus on:
- Is the problem statement clear and evidence-based, or is it assumed?
- Are success metrics measurable and realistic?
- Is the scope right-sized? Flag gold-plating AND under-specification.
- Are there gaps in the user stories? Missing edge cases? Missing personas?
- Are the constraints sufficient or do they leave dangerous ambiguity?
- Is the "What If" list honest? What scenarios are missing?
- Are the open questions the RIGHT questions, or are they dodging harder ones?
- Would you approve this PRD to move to tech spec? If not, what's blocking?

Be direct, not diplomatic. If something is fine but would cause regret
in 6 months, say so now. If the scope is too vague to implement, block it.

---

{{doc_content}}`,

	TechSpec: `Review the following technical specification as a principal engineer.

Ask the hard questions:
- What happens at 10x the expected load?
- What's the failure mode? What breaks first?
- What's the migration path from current state?
- Are there coupling decisions that will be expensive to reverse?
- Is this over-engineered for the current scale?
- Is this under-engineered for where we're heading?
- Are there non-obvious downstream consequences of pattern choices?
- Is the testing strategy sufficient? What's untested?

Focus on: correctness, edge cases, performance cliffs, security, naming.
Don't nitpick formatting. If something is fine but fragile, flag it.

---

{{doc_content}}`,
}

// DecomposePrompt is the template for story decomposition from a spec.
// Contains a {{spec_content}} placeholder.
const DecomposePrompt = `Decompose the following spec into implementable stories.

Rules:
- Each story must be independently testable
- Each story must have clear, specific acceptance criteria
- If acceptance criteria need a paragraph, the story is too big — split it
- Order stories by dependency (what must exist before what)
- Group into phases where stories in the same phase have no inter-dependencies
- Flag any trade-offs in the decomposition
- Identify blocked_by relationships explicitly
- Each story should be completable in a single Claude Code session

Present options if there are multiple valid decomposition strategies.
Don't pick for me — show the trade-offs.

---

{{spec_content}}`
