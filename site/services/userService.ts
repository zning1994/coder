import { createMachine, interpret } from "xstate"
import * as API from "../api"

export interface User {
  readonly id: string
  readonly username: string
  readonly email: string
  readonly created_at: string
}

export interface UserContext {
  readonly error?: Error
  readonly me?: User
  redirectOnError: boolean
  shouldRedirect?: boolean
}

const signOut = async () => {
  await API.logout()
}

const userMachine = 
/** @xstate-layout N4IgpgJg5mDOIC5QFdZgE4GUAuBDbYAdLAJZQB2kA8stgMSYCSA4gHID6jrioADgPalsJfuR4gAHogAcAJkIBGAJxKArAoDMqpRoAsAdn0bZAGhABPRBulLCs2XLUaNABk0vVAX09nUGHPhEpBQk5FCM5HQQokShAG78ANZBZOQR4gJCImJIklYu8voFctKqGgoAbPraZpYIFdIudm66SroVeqoOFd6+aFh4BMSpoeGRGOj86IS8ADb4AGZTALbDFOm5mSTCouJSCHryunpKLrrHurL6yrWIZRWEFQpuJRr6ci76vSB+A4FrlAgEUIJAgszAdAAogARRgAFQygm22T2iAqSgeCme1zKXSUCmOt3qskxBPUBP0bUaum+vwCQ2CgOBkGRYSiMRB5ASyUILOwAFV+oisrtcvtpO1CNZdAp9LoPIdVBUiaoPIoye53tJZQTaf16SkKJAIgwWBwqPyEZskTscqB9uV9IptdI5Zp1KVqkTnqpCDpPm9WspjNY9f5BobyKMaPRopROdzIzHhcjRfbELJTs15a62mp5R4ibJVLpCDYVApVNdLsoHGG-gyRmEY3QJlMZvNsEt0KtGcnrSK7Xl6hUHi5tRV7LICy50d61QobHKMWS3IvvD4QOR+BA4OI6RGAdRaCnbaj6k0XEpZHoio0tG05fPfQVKq4NPj0RUCvWDQDRhsfA2iiYoZk6KiqK6zhVKqzymBYdxaIQBjLlcug2AS0i-oejLGuQIJgmAp4gemCBVtI6oBtqlZKtIyoIcSFFYro6hnMoLGtNh-y4UC+F8qMxFpsOlK2Bo35ysWWp6AoRKTk0zHkjehiyD+m4HtxqR4YJQ77KoJaUUY1F6Q09F1Mc8gKTmJzHFhan6jhTZQP2QGDuejROrK7Qfm8s5vEWFSlsWj5FLKrpeHZ4aBNp54aH6ahQWJ1RuG4bREsYhCvh0V6GCW0gaBunhAA */
createMachine(
  {
  id: "userState",
  initial: "signedOut",
  states: {
    signedOut: {
      on: {
        SIGN_IN: {
          target: "#userState.signingIn",
        },
      },
    },
    signingIn: {
      invoke: {
        src: "signIn",
        id: "signIn",
        onDone: [
          {
            actions: "assignMe",
            target: "#userState.signedIn",
          },
        ],
        onError: [
          {
            actions: "assignError",
            target: "#userState.signedOut",
          },
        ],
      },
    },
    signedIn: {
      states: {
        idle: {
          on: {
            EDIT: {
              target: "#userState.signedIn.editing",
            },
          },
        },
        editing: {
          invoke: {
            src: "editUser",
            id: "editUser",
            onDone: [
              {
                actions: "updateUser",
                target: "#userState.signedIn.idle",
              },
            ],
          },
        },
      },
      on: {
        SIGN_OUT: {
          target: "#userState.signingOut",
        },
      },
    },
    signingOut: {
      invoke: {
        src: "signOut",
        id: "signOut",
        onDone: [
          {
            actions: "unassignMe",
            target: "#userState.signedOut",
          },
        ],
        onError: [
          {
            actions: ["assignError", "assignRedirect"],
            target: "#userState.signedOut",
          },
        ],
      },
    },
  },
},
  {
    services: {
      signOut
    }
  }
)

export const userService = interpret(userMachine).start()