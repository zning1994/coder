// This file was automatically generated. Edits will be overwritten

export interface Typegen0 {
  "@@xstate/typegen": true
  eventsCausingActions: {
    saveData: "_FETCH_SUCCEEDED"
    saveError: "_FETCH_FAILED" | "_FETCH_STARTED"
    mutate: "REFETCH"
  }
  internalEvents: {
    "xstate.init": { type: "xstate.init" }
  }
  invokeSrcNameMap: {}
  missingImplementations: {
    actions: never
    services: never
    guards: never
    delays: never
  }
  eventsCausingServices: {}
  eventsCausingGuards: {}
  eventsCausingDelays: {}
  matchesStates: "pending" | "fetching" | "fetched" | "revalidating" | "stale" | "failed"
  tags: never
}
