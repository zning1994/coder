import { KeyedMutator, mutate, MutatorOptions } from "swr";
import { createMachine, assign } from "xstate";

export interface SWRContext {
  data?: unknown,
  error?: Error
}

export type SWREvent = 
  | { type: '_FETCH_STARTED', error?: Error }
  | { type: '_FETCH_SUCCEEDED', data: unknown }
  | { type: '_FETCH_FAILED', error: Error }
  | { type: 'REFETCH', data?: unknown, opts?: boolean | MutatorOptions<unknown> | undefined }

export const makeSwrMachine = (boundMutate: KeyedMutator<unknown>) => createMachine({
  tsTypes: {} as import("./swrMachine.typegen").Typegen0,
  schema: {
    context: { data: undefined, error: undefined } as SWRContext,
    events: {} as SWREvent
  },
    id: `swr`,
    initial: "pending",
    states: {
      pending: {
        on: { _FETCH_STARTED: "fetching" }
      },
      fetching: {
        on: {
          _FETCH_SUCCEEDED: {
            target: "fetched",
            actions: "saveData"
          },
          _FETCH_FAILED: {
            target: "stale",
            actions: "saveError"
          }
        }
      },
      fetched: {
        on: {
          _FETCH_STARTED: "revalidating",
          REFETCH: { actions: ["mutate"] }
        }
      },
      revalidating: {
        on: {
          _FETCH_SUCCEEDED: {
            target: "fetched",
            actions: "saveData"
          },
          _FETCH_FAILED: {
            target: "stale",
            actions: "saveError"
          }
        }
      },
      stale: {
        on: {
          _FETCH_STARTED: {
            target: "revalidating",
            actions: "saveError"
          },
          REFETCH: { actions: ["mutate"] }
        }
      },
      failed: {
        on: {
          _FETCH_STARTED: {
            target: "revalidating",
            actions: "saveError"
          },
          REFETCH: { actions: ["mutate"] }
        }
      }
    }
  },
  {
    actions: {
      saveData: assign({
        data: (_, event) => event.data
      }),
      saveError: assign({
        error: (_, event) => event.error
      }),
      mutate: (_, event: SWREvent) => {
        if (event.type === 'REFETCH') {
          boundMutate(event.data, event.opts)
        }
      }
    }
  }
);
