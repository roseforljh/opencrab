# Graph Report - .  (2026-04-27)

## Corpus Check
- 160 files ¡¤ ~142,852 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1528 nodes ¡¤ 2503 edges ¡¤ 79 communities detected
- Extraction: 100% EXTRACTED ¡¤ 0% INFERRED ¡¤ 0% AMBIGUOUS
- Token cost: 0 input ¡¤ 0 output

## God Nodes (most connected - your core abstractions)
1. `roundTripFunc` - 41 edges
2. `testGatewayMessage()` - 40 edges
3. `testGatewayMessage()` - 25 edges
4. `decodeJSONObject()` - 23 edges
5. `newGatewayServiceForTest()` - 22 edges
6. `GetDashboardSummary()` - 18 edges
7. `renderGatewayErrorForProtocol()` - 16 edges
8. `sanitizeRequestMetadataForTarget()` - 15 edges
9. `writeGatewayResult()` - 15 edges
10. `writeResponsesGatewayResult()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `PUT()` --calls--> `proxy()`  [EXTRACTED]
  web\src\app\api\admin\[...path]\route.ts ¡ú web\src\app\v1beta\[...path]\route.ts
- `DELETE()` --calls--> `proxy()`  [EXTRACTED]
  web\src\app\api\admin\[...path]\route.ts ¡ú web\src\app\v1beta\[...path]\route.ts
- `proxy()` --calls--> `buildUpstreamUrl()`  [EXTRACTED]
  web\src\app\v1beta\[...path]\route.ts ¡ú web\src\app\api\auth\[action]\route.ts

## Communities

### Community 0 - "Community 0"
Cohesion: 0.02
Nodes (27): formatCompactNumber(), formatLatency(), formatNumber(), deleteOne(), handleDelete(), createEmptyDraft(), draftFromItem(), handleDelete() (+19 more)

### Community 1 - "Community 1"
Cohesion: 0.05
Nodes (102): acceptGatewayRequest(), applyClaudeContextManagement(), buildAcceptedResponse(), buildGatewayJobStatusResponse(), buildStoredGatewayRequest(), clearHistoricalThinking(), clearHistoricalToolUses(), cloneHeaderMap() (+94 more)

### Community 2 - "Community 2"
Cohesion: 0.06
Nodes (77): applyRequestHeaders(), asInt(), attachPayloadDebugMetadata(), buildExecutorPayload(), buildFocusedPayloadPreview(), buildToolNameReverseMap(), claudeBudgetToOpenAIEffort(), claudeToolPartToGeminiFunctionResponse() (+69 more)

### Community 3 - "Community 3"
Cohesion: 0.06
Nodes (62): applyDispatchSettingsSummary(), boolToInt(), buildDashboardChannelMix(), buildDashboardDailyCounts(), buildDashboardModelRanking(), buildDashboardRecentLogs(), buildDashboardShareItems(), buildDashboardTrafficSeries() (+54 more)

### Community 4 - "Community 4"
Cohesion: 0.03
Nodes (0): 

### Community 5 - "Community 5"
Cohesion: 0.04
Nodes (31): testHTTPClient(), TestKimiRealChannel(), TestOpenRouterRealChannel(), buildGeminiCachedContentResponse(), decodeGeminiCachedContentName(), HandleGeminiCachedContentCreate(), HandleGeminiCachedContentGet(), mustMarshalJSON() (+23 more)

### Community 6 - "Community 6"
Cohesion: 0.04
Nodes (5): captureHandler, captureLogsContain(), newCaptureLogger(), TestProxyClaudeMessagesSynthesizesClaudeStreamFromOpenAIResponse(), TestProxyResponsesLogsRenderedProxyWriteFailureWithoutOverwritingStatus()

### Community 7 - "Community 7"
Cohesion: 0.09
Nodes (38): newGatewayServiceForTest(), stickyKey(), testGatewayMessage(), TestGatewayServiceAllAttemptsFailedAfterThreeRetriesPerRoute(), TestGatewayServiceAnnotatesSuccessfulResultWithTargetOperation(), TestGatewayServiceBasicResponsesRequestCanBridgeToClaude(), TestGatewayServiceClaudeNativeHeadersRequireClaudeProvider(), TestGatewayServiceClaudeNativeMetadataRequireClaudeProvider() (+30 more)

### Community 8 - "Community 8"
Cohesion: 0.1
Nodes (43): gjsonToolCalls(), TestClaudeExecutorBuildsMultimodalRequest(), TestClaudeExecutorBuildsNativeTextRequest(), TestClaudeExecutorPassesThroughAnthropicBetaHeader(), TestClaudeExecutorTransformsOpenAICodeInterpreterToolToContainer(), TestClaudeExecutorTransformsOpenAIMCPToolAndAddsBetaHeader(), TestClaudeExecutorTransformsOpenAIReasoningToThinking(), testGatewayMessage() (+35 more)

### Community 9 - "Community 9"
Cohesion: 0.05
Nodes (43): AdminAuthState, AdminAuthStatus, AdminPasswordChangeInput, AdminPasswordInput, AdminSecondaryPasswordUpdateInput, AdminSecondarySecurityState, APIKey, APIKeyScope (+35 more)

### Community 10 - "Community 10"
Cohesion: 0.05
Nodes (32): ChatCompletionsMessage, ChatCompletionsRequest, ChatProvider, ExecutionError, ExecutionResult, Executor, ExecutorRequest, GatewayAcceptedResponse (+24 more)

### Community 11 - "Community 11"
Cohesion: 0.1
Nodes (40): appendResponsesItemEvents(), appendResponsesMessageEvents(), appendResponsesReasoningEvents(), appendResponsesStatusEvents(), appendResponsesStringFieldEvents(), BuildOpenAIResponsesEvents(), DecodeOpenAIResponsesRequest(), DecodeOpenAIResponsesResponse() (+32 more)

### Community 12 - "Community 12"
Cohesion: 0.11
Nodes (22): adaptRequestForProvider(), buildRoutingCursorKey(), groupRoutesByPriority(), invocationBucketName(), marshalGatewayRequest(), nativePreferredProvider(), preferNativeProviderRoutes(), prioritizeRoutesByInvocationMode() (+14 more)

### Community 13 - "Community 13"
Cohesion: 0.08
Nodes (18): applyAPIKeyScopeToGatewayRequest(), extractGatewayAPIKey(), extractGatewaySessionID(), extractModel(), extractModelFromRequest(), extractSessionAffinityKey(), extractSSEUsageMetrics(), extractStringRawValue() (+10 more)

### Community 14 - "Community 14"
Cohesion: 0.1
Nodes (8): adminFetch(), getAdminAuthStatus(), getAdminDashboardSummary(), getAdminLogDetail(), getAdminLogs(), getAdminRoutingOverview(), getAdminSecondarySecurityState(), resolveAdminFetchFailure()

### Community 15 - "Community 15"
Cohesion: 0.15
Nodes (25): buildClaudeSource(), buildClaudeToolResultBlocks(), claudeThinkingSupported(), claudeToolChoiceDisallowsThinking(), claudeTopLevelAndNestedCacheTTL(), collectClaudeCacheControlTTLs(), collectClaudeCacheControlTTLsInto(), containsCacheTTL() (+17 more)

### Community 16 - "Community 16"
Cohesion: 0.15
Nodes (26): decodeGeminiCandidate(), DecodeGeminiChatRequest(), DecodeGeminiChatResponse(), DecodeGeminiChatStream(), decodeGeminiContent(), decodeGeminiFunctionResponseText(), decodeGeminiMessages(), decodeGeminiPart() (+18 more)

### Community 17 - "Community 17"
Cohesion: 0.16
Nodes (21): buildChatCompletionsURL(), buildClaudeMessagesURL(), BuildExecutorPayload(), buildGeminiGenerateContentURL(), buildGeminiStreamGenerateContentURL(), buildResponsesURL(), DecodeUpstreamResponse(), DecodeUpstreamResponseForOperation() (+13 more)

### Community 18 - "Community 18"
Cohesion: 0.12
Nodes (12): App, buildSystemSettingGroups(), containsString(), New(), AppConfig, Config, DBConfig, getEnv() (+4 more)

### Community 19 - "Community 19"
Cohesion: 0.1
Nodes (5): GatewayRuntimeConfigStore, RoutingConfigStore, RoutingCursorStore, RoutingRuntimeStateStore, StickyRoutingStore

### Community 20 - "Community 20"
Cohesion: 0.28
Nodes (19): collectUnknownFields(), DecodeOpenAIChatRequest(), DecodeOpenAIChatResponse(), decodeOpenAIMessageContent(), decodeOpenAIPart(), decodeOpenAIResponseMessage(), decodeOpenAIToolCalls(), decodePreservedMapItem() (+11 more)

### Community 21 - "Community 21"
Cohesion: 0.13
Nodes (7): signedSessionCookieValue(), TestCapabilityProfilesDeleteAcceptsPayload(), TestCapabilityProfilesListReturnsCatalog(), TestCapabilityProfilesUpdateAcceptsPayload(), TestCreateAPIKeyRejectsMissingSecondaryPasswordWhenEnabled(), TestDeleteAPIKeyAcceptsSecondaryPasswordHeader(), TestSettingsRejectAdminSecurityKeys()

### Community 22 - "Community 22"
Cohesion: 0.18
Nodes (14): buildGeminiCachedContentCreateURL(), buildGeminiCachedContentGetURL(), buildGeminiStreamGenerateContentURL(), buildOpenAIRealtimeCallsURL(), buildOpenAIRealtimeClientSecretsURL(), buildOpenAIRealtimeWebSocketURL(), DialOpenAIRealtime(), doProxyRequest() (+6 more)

### Community 23 - "Community 23"
Cohesion: 0.11
Nodes (0): 

### Community 24 - "Community 24"
Cohesion: 0.26
Nodes (12): realtimeConnectionState, realtimeWebSocketEnvelope, buildRealtimeConversationEvent(), decodeOptionalRawMap(), decodeRawMap(), decodeRawStringValue(), HandleOpenAIRealtime(), marshalRawArray() (+4 more)

### Community 25 - "Community 25"
Cohesion: 0.21
Nodes (3): isUniqueConstraintError(), nowRFC3339(), GatewayJobStore

### Community 26 - "Community 26"
Cohesion: 0.13
Nodes (4): MemoryResponseSessionStore, ResponseSession, ResponseSessionStore, ResponseSessionStore

### Community 27 - "Community 27"
Cohesion: 0.21
Nodes (7): buildAdminAuthStatus(), clearAdminSessionCookie(), hasValidAdminSession(), requestIsSecure(), requireAdminSession(), signAdminSession(), writeAdminSessionCookie()

### Community 28 - "Community 28"
Cohesion: 0.26
Nodes (9): buildClaudeMessagesURL(), buildClaudeTestRequest(), buildGeminiGenerateContentURL(), buildGeminiTestRequest(), buildOpenAICompatibleTestRequest(), buildTestRequest(), defaultTestModel(), normalizeProvider() (+1 more)

### Community 29 - "Community 29"
Cohesion: 0.22
Nodes (8): cloneRawMap(), decodeStringRaw(), enrichPartMetadata(), extractMediaDescriptor(), formatToMime(), rawJSONString(), setRawString(), mediaDescriptor

### Community 30 - "Community 30"
Cohesion: 0.15
Nodes (0): 

### Community 31 - "Community 31"
Cohesion: 0.35
Nodes (11): capabilitySatisfiedByProvider(), claudeMCPServersCanBridgeToOpenAI(), decodeCapabilityRawString(), EvaluateGatewayRoute(), has(), openAIMCPToolsCanBridgeToClaude(), resolveOperationSurface(), resolveTargetOperation() (+3 more)

### Community 32 - "Community 32"
Cohesion: 0.17
Nodes (9): GatewayError, Protocol, ProtocolOperation, UnifiedChatRequest, UnifiedChatResponse, UnifiedMessage, UnifiedPart, UnifiedStreamEvent (+1 more)

### Community 33 - "Community 33"
Cohesion: 0.21
Nodes (5): buildChatCompletionsURL(), CopyStreamResponse(), ForwardChatCompletions(), isSSEHeader(), OpenAICompatibleProvider

### Community 34 - "Community 34"
Cohesion: 0.36
Nodes (11): appendJSON(), appendMessagesWindow(), cloneUnifiedMessage(), cloneUnifiedToolCalls(), findTrailingPendingToolExchangeStartUnified(), leadingSystemMessageCount(), normalizeResponsesTextPart(), projectOpenAIResponsesRequest() (+3 more)

### Community 35 - "Community 35"
Cohesion: 0.32
Nodes (11): ChangeAdminPassword(), generateAdminSessionSecret(), GetAdminAuthState(), GetAdminSecondarySecurityState(), getAdminSecondaryState(), SetupAdminPassword(), UpdateAdminSecondaryPassword(), VerifyAdminPassword() (+3 more)

### Community 36 - "Community 36"
Cohesion: 0.35
Nodes (10): analyzeClaudeMetadata(), AnalyzeGatewayRequest(), analyzeGeminiMetadata(), analyzeMessageParts(), analyzeOpenAIMetadata(), analyzeRawTool(), analyzeTools(), decodeRawToolType() (+2 more)

### Community 37 - "Community 37"
Cohesion: 0.27
Nodes (10): BuildExecutionPlan(), buildTransformPlan(), normalizeSourceOperation(), PlanRoute(), targetProtocolForProvider(), AttemptPlan, ExecutionPlan, HopMode (+2 more)

### Community 38 - "Community 38"
Cohesion: 0.31
Nodes (6): max(), max64(), normalizeDispatchLimit(), normalizedReservationKey(), toInt64(), RedisDispatchQuotaManager

### Community 39 - "Community 39"
Cohesion: 0.2
Nodes (0): 

### Community 40 - "Community 40"
Cohesion: 0.2
Nodes (0): 

### Community 41 - "Community 41"
Cohesion: 0.28
Nodes (6): Loader, ProfileRecord, Registry, ScopeType, applyProfileRecord(), cloneCapabilitySet()

### Community 42 - "Community 42"
Cohesion: 0.28
Nodes (4): NewGatewayAttemptLogStore(), NewRequestLogStore(), GatewayAttemptLogStore, RequestLogStore

### Community 43 - "Community 43"
Cohesion: 0.25
Nodes (0): 

### Community 44 - "Community 44"
Cohesion: 0.36
Nodes (5): normalizeRejectCode(), rejectMessage(), rejectStatusCode(), Decision, Engine

### Community 45 - "Community 45"
Cohesion: 0.29
Nodes (3): normalizeCapabilityList(), capabilityProfileConfig, CapabilityProfileStore

### Community 46 - "Community 46"
Cohesion: 0.46
Nodes (7): extractRealtimeModelFromBody(), extractRealtimeModelFromJSON(), extractRealtimeModelFromMultipart(), HandleOpenAIRealtimeCalls(), HandleOpenAIRealtimeClientSecrets(), maybeProxyOpenAIRealtimeWebSocket(), proxyRealtimeSockets()

### Community 47 - "Community 47"
Cohesion: 0.25
Nodes (1): fakeDispatchRuntimeConfigStore

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (3): Capability, RequestProfile, RouteCompatibility

### Community 49 - "Community 49"
Cohesion: 0.29
Nodes (6): DispatchQuotaManager, DispatchReleaseInput, DispatchReservationInput, DispatchReservationResult, DispatchRuntimeConfigStore, DispatchRuntimeSettings

### Community 50 - "Community 50"
Cohesion: 0.43
Nodes (5): EncodeClaudeChatStream(), EncodeOpenAIChatStream(), firstUsageValue(), mustClaudeSSE(), mustOpenAISSE()

### Community 51 - "Community 51"
Cohesion: 0.52
Nodes (6): buildUpstreamUrl(), DELETE(), GET(), POST(), proxy(), PUT()

### Community 52 - "Community 52"
Cohesion: 0.6
Nodes (5): appendRealtimeOutputEvents(), BuildOpenAIRealtimeEvents(), buildRealtimeConversationItemEvent(), buildRealtimeOutputItemEvent(), cloneRealtimeItem()

### Community 53 - "Community 53"
Cohesion: 0.47
Nodes (1): GatewayStore

### Community 54 - "Community 54"
Cohesion: 0.33
Nodes (0): 

### Community 55 - "Community 55"
Cohesion: 0.67
Nodes (5): rawMapHasKeys(), requestRequiresProtocolMatchedRoute(), requestUsesClaudeNativeFeatures(), requestUsesGeminiNativeFeatures(), requestUsesTools()

### Community 56 - "Community 56"
Cohesion: 0.4
Nodes (2): limiterEntry, RateLimiter

### Community 57 - "Community 57"
Cohesion: 0.5
Nodes (4): providerMatrix, surfaceSupport, capabilitySet(), providerSupportMatrix()

### Community 58 - "Community 58"
Cohesion: 0.5
Nodes (3): fakeLoader, boolPtr(), TestRegistryMergesProviderChannelAndModelOverrides()

### Community 59 - "Community 59"
Cohesion: 0.4
Nodes (0): 

### Community 60 - "Community 60"
Cohesion: 0.5
Nodes (2): parseDispatchBool(), DispatchRuntimeConfigStore

### Community 61 - "Community 61"
Cohesion: 0.4
Nodes (0): 

### Community 62 - "Community 62"
Cohesion: 0.5
Nodes (0): 

### Community 63 - "Community 63"
Cohesion: 0.67
Nodes (0): 

### Community 64 - "Community 64"
Cohesion: 0.67
Nodes (0): 

### Community 65 - "Community 65"
Cohesion: 0.67
Nodes (0): 

### Community 66 - "Community 66"
Cohesion: 1.0
Nodes (2): ApplyMigrations(), hasMigration()

### Community 67 - "Community 67"
Cohesion: 1.0
Nodes (0): 

### Community 68 - "Community 68"
Cohesion: 1.0
Nodes (0): 

### Community 69 - "Community 69"
Cohesion: 1.0
Nodes (0): 

### Community 70 - "Community 70"
Cohesion: 1.0
Nodes (0): 

### Community 71 - "Community 71"
Cohesion: 1.0
Nodes (0): 

### Community 72 - "Community 72"
Cohesion: 1.0
Nodes (0): 

### Community 73 - "Community 73"
Cohesion: 1.0
Nodes (0): 

### Community 74 - "Community 74"
Cohesion: 1.0
Nodes (0): 

### Community 75 - "Community 75"
Cohesion: 1.0
Nodes (0): 

### Community 76 - "Community 76"
Cohesion: 1.0
Nodes (0): 

### Community 77 - "Community 77"
Cohesion: 1.0
Nodes (0): 

### Community 78 - "Community 78"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **126 isolated node(s):** `surfaceSupport`, `providerMatrix`, `ScopeType`, `ProfileRecord`, `Loader` (+121 more)
  These have ¡Ü1 connection - possible missing edges or undocumented components.
- **Thin community `Community 67`** (2 nodes): `capability_store_test.go`, `TestCapabilityProfileStoreListCapabilityProfiles()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 68`** (2 nodes): `db.go`, `Open()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (2 nodes): `migrate_test.go`, `TestApplyMigrations()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (2 nodes): `response_session_store_test.go`, `TestResponseSessionStoreRoundTrip()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (2 nodes): `not-found.tsx`, `NotFound()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 72`** (2 nodes): `loading.tsx`, `ConsoleLoading()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (2 nodes): `template.tsx`, `ConsoleTemplate()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (2 nodes): `loading-state.tsx`, `LoadingState()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (2 nodes): `admin-api-server.test.ts`, `assert()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (1 nodes): `next-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (1 nodes): `tailwind.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (1 nodes): `console-data.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What connects `surfaceSupport`, `providerMatrix`, `ScopeType` to the rest of the system?**
  _126 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.02 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.05 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._
- **Should `Community 4` be split into smaller, more focused modules?**
  _Cohesion score 0.03 - nodes in this community are weakly interconnected._
- **Should `Community 5` be split into smaller, more focused modules?**
  _Cohesion score 0.04 - nodes in this community are weakly interconnected._