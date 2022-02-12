import React from "react";
import useSWR, { Fetcher, Key } from "swr";
import { useMachine } from "@xstate/react";
import { makeSwrMachine } from "../machines/swrMachine";

export const useSWRMachine = (key: Key, fetcher: Fetcher) => {
  const { data, error, isValidating, mutate } = useSWR(key, fetcher);

  const [state, send] = useMachine(
    makeSwrMachine(mutate)
  );

  React.useEffect(() => {
    send("_FETCH_STARTED");
  }, [send]);

  React.useEffect(() => {
    if (isValidating) {
      send("_FETCH_STARTED");
    } else if (error) {
      send("_FETCH_FAILED", { error });
    } else if (data) {
      send("_FETCH_SUCCEEDED", { data });
    }
  }, [isValidating, send, error, data]);

  return [state, send];
};
