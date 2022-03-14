import { assign } from "xstate/lib/actions"
import * as API from "../../api"
import { UserEvent } from "./userService"

export const services = {
  getMe: async () => {
    await API.getUser()
  },

  signIn: async (_, event: UserEvent) => {
    await API.login(event.email, event.password)
  },

  signOut: async () => {
    await API.logout()
  },
}

export const actions = {
  assignMe: assign({
    me: (context, event) => event.data
  }),

  unassignMe: assign({
    me: () => undefined,
  }),

  assignError: assign({
    error: (context, event) => event.data,
  }),
}
