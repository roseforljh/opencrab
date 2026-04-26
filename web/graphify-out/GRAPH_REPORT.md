# Graph Report - .  (2026-04-27)

## Corpus Check
- 77 files ¡¤ ~27,995 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 200 nodes ¡¤ 313 edges ¡¤ 17 communities detected
- Extraction: 100% EXTRACTED ¡¤ 0% INFERRED ¡¤ 0% AMBIGUOUS
- Token cost: 0 input ¡¤ 0 output

## God Nodes (most connected - your core abstractions)
1. `adminFetch()` - 7 edges
2. `proxy()` - 6 edges
3. `resetDraft()` - 4 edges
4. `handleEdit()` - 3 edges
5. `selectDisplayLogs()` - 3 edges
6. `isNativeDirectLog()` - 3 edges
7. `normalizeAliasDraft()` - 3 edges
8. `formatNumber()` - 3 edges
9. `deleteOne()` - 2 edges
10. `handleDelete()` - 2 edges

## Surprising Connections (you probably didn't know these)
- `PUT()` --calls--> `proxy()`  [EXTRACTED]
  src\app\api\admin\[...path]\route.ts ¡ú src\app\v1beta\[...path]\route.ts
- `DELETE()` --calls--> `proxy()`  [EXTRACTED]
  src\app\api\admin\[...path]\route.ts ¡ú src\app\v1beta\[...path]\route.ts
- `proxy()` --calls--> `buildUpstreamUrl()`  [EXTRACTED]
  src\app\v1beta\[...path]\route.ts ¡ú src\app\api\auth\[action]\route.ts

## Communities

### Community 0 - "Community 0"
Cohesion: 0.09
Nodes (0): 

### Community 1 - "Community 1"
Cohesion: 0.08
Nodes (6): buildPath(), buildPoints(), isSuccessStatus(), StatusPill(), describePressure(), RuntimePressureGauge()

### Community 2 - "Community 2"
Cohesion: 0.13
Nodes (9): formatCompactNumber(), formatLatency(), formatNumber(), buildRoutingNarrative(), isNativeDirectLog(), matchesLogCategory(), parseLogDetails(), scoreLog() (+1 more)

### Community 3 - "Community 3"
Cohesion: 0.12
Nodes (3): handleCreateAlias(), handleUpdate(), normalizeAliasDraft()

### Community 4 - "Community 4"
Cohesion: 0.11
Nodes (2): deleteOne(), handleDelete()

### Community 5 - "Community 5"
Cohesion: 0.15
Nodes (0): 

### Community 6 - "Community 6"
Cohesion: 0.17
Nodes (0): 

### Community 7 - "Community 7"
Cohesion: 0.23
Nodes (7): adminFetch(), getAdminAuthStatus(), getAdminDashboardSummary(), getAdminLogDetail(), getAdminLogs(), getAdminRoutingOverview(), getAdminSecondarySecurityState()

### Community 8 - "Community 8"
Cohesion: 0.36
Nodes (7): createEmptyDraft(), draftFromItem(), handleDelete(), handleEdit(), handleSave(), keyOf(), resetDraft()

### Community 9 - "Community 9"
Cohesion: 0.52
Nodes (6): buildUpstreamUrl(), DELETE(), GET(), POST(), proxy(), PUT()

### Community 10 - "Community 10"
Cohesion: 1.0
Nodes (0): 

### Community 11 - "Community 11"
Cohesion: 1.0
Nodes (0): 

### Community 12 - "Community 12"
Cohesion: 1.0
Nodes (0): 

### Community 13 - "Community 13"
Cohesion: 1.0
Nodes (0): 

### Community 14 - "Community 14"
Cohesion: 1.0
Nodes (0): 

### Community 15 - "Community 15"
Cohesion: 1.0
Nodes (0): 

### Community 16 - "Community 16"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **Thin community `Community 10`** (2 nodes): `not-found.tsx`, `NotFound()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 11`** (2 nodes): `loading.tsx`, `ConsoleLoading()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 12`** (2 nodes): `template.tsx`, `ConsoleTemplate()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 13`** (2 nodes): `loading-state.tsx`, `LoadingState()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 14`** (1 nodes): `next-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 15`** (1 nodes): `tailwind.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 16`** (1 nodes): `console-data.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.09 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.08 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.13 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.12 - nodes in this community are weakly interconnected._
- **Should `Community 4` be split into smaller, more focused modules?**
  _Cohesion score 0.11 - nodes in this community are weakly interconnected._