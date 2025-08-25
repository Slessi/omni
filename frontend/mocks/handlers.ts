import { http, HttpResponse } from 'msw'

export const handlers = [
  http.post('/omni.resources.ResourceService/Watch', () => {
    return HttpResponse.json({
      id: 'abc-123',
      firstName: 'John',
      lastName: 'Maverick',
    })
  }),
]
