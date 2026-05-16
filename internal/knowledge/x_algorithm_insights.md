# X Algorithm Writing Insights

Source snapshot: xai-org/x-algorithm README and phoenix/README, checked 2026-05-16.

Use this as a writing heuristic for X/Twitter drafts. It is not live access to X
ranking, not a guarantee of reach, and not a replacement for audience-specific
analytics.

## Ranking Model To Keep In Mind

- The For You feed combines in-network content from followed accounts with
  out-of-network content discovered by Phoenix retrieval.
- Candidate posts pass through hydration, filtering, scoring, selection, and a
  final filtering pass before appearing in the feed.
- Phoenix retrieval narrows a large corpus to likely candidates, then Phoenix
  ranking predicts engagement probabilities for each candidate.
- The final score is a weighted mix of predicted actions. Positive actions push
  a post up, while negative actions push it down.
- Repeated author exposure can be attenuated for diversity, so every post should
  stand alone instead of relying only on author familiarity.

## Positive Signals

Drafts should increase the chance of useful actions:

- like or favorite
- reply
- repost
- quote
- click
- profile click
- media expand or video view
- share
- dwell
- follow author

## Negative Signals

Drafts should reduce the chance of rejection signals:

- not interested
- block author
- mute author
- report
- muted-keyword or low-quality-topic reactions

## Dispatch Writing Rubric

When drafting X/Twitter posts or threads, optimize for these checks:

- Hook: the first sentence should make the post's specific developer payoff
  obvious without clickbait.
- Dwell: add a compact story, contrast, code insight, or surprising detail that
  rewards reading to the end.
- Reply: invite a concrete response with a real tradeoff, decision, example, or
  question. Avoid generic "thoughts?" prompts.
- Repost: make the post useful enough to pass along: a sharp lesson, checklist,
  comparison, principle, or repeatable workflow.
- Click/profile-follow intent: make expertise visible through specificity, not
  hype.
- Risk: remove vague claims, engagement bait, outrage framing, repeated wording,
  and anything likely to trigger mute, block, report, or not-interested signals.

For X/Twitter previews, include a short visible rubric with exactly these labels:
Hook, Dwell, Reply, Repost, Risiko.
