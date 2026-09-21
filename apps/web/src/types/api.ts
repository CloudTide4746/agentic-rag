export interface ApiResponse<T> { data: T; request_id: string }
export interface ApiErrorResponse { error: { code: string; message: string }; request_id: string }
export interface Health { status: string; service: string }
