import { http, HttpResponse } from "msw"

export const handlers = [
  http.get("/api/v1/auth/me", () => {
    return HttpResponse.json({
      data: {
        id: "test-user-id",
        nickname: "TestPlayer",
        avatar_url: "https://example.com/avatar.png",
        faceit_elo: 2100,
        faceit_level: 9,
        country: "US",
      },
    })
  }),
]
