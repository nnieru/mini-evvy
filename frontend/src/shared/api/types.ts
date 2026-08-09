import type { components } from './schema'

export type User = components['schemas']['User']
export type AuthResponse = components['schemas']['AuthResponse']
export type Organization = components['schemas']['Organization']
export type Member = components['schemas']['Member']
export type MyRole = components['schemas']['MyRole']
export type Event = components['schemas']['Event']
export type MyEvent = components['schemas']['MyEvent']
export type SeatCategory = components['schemas']['SeatCategory']
export type Seat = components['schemas']['Seat']
export type Guest = components['schemas']['Guest']
export type Booking = components['schemas']['Booking']
export type BookingListItem = components['schemas']['BookingListItem']
export type PaginatedBookingList = components['schemas']['PaginatedBookingList']
export type PaginatedGuestList = components['schemas']['PaginatedGuestList']
export type PaginatedSeatList = components['schemas']['PaginatedSeatList']
export type PaginatedAttendanceList = components['schemas']['PaginatedAttendanceList']
export type PaginatedJobList = components['schemas']['PaginatedJobList']
export type Payment = components['schemas']['Payment']
export type Attendance = components['schemas']['Attendance']
export type Job = components['schemas']['Job']
export type JobIdResponse = components['schemas']['JobIdResponse']

export type RegisterRequest = components['schemas']['RegisterRequest']
export type LoginRequest = components['schemas']['LoginRequest']
export type CreateOrgRequest = components['schemas']['CreateOrgRequest']
export type AddMemberRequest = components['schemas']['AddMemberRequest']
export type CreateEventRequest = components['schemas']['CreateEventRequest']
export type UpdateEventRequest = components['schemas']['UpdateEventRequest']
export type CreateCategoryRequest = components['schemas']['CreateCategoryRequest']
export type UpdateCategoryRequest = components['schemas']['UpdateCategoryRequest']
export type CreateSeatItem = components['schemas']['CreateSeatItem']
export type CreateSeatsRequest = components['schemas']['CreateSeatsRequest']
export type UpdateSeatRequest = components['schemas']['UpdateSeatRequest']
export type CreateGuestRequest = components['schemas']['CreateGuestRequest']
export type UpdateGuestRequest = components['schemas']['UpdateGuestRequest']
export type CreateBookingRequest = components['schemas']['CreateBookingRequest']
export type CreateBookingBatchRequest = components['schemas']['CreateBookingBatchRequest']
export type UpdateBookingRequest = components['schemas']['UpdateBookingRequest']
export type CreatePaymentRequest = components['schemas']['CreatePaymentRequest']
export type CreateAttendanceRequest = components['schemas']['CreateAttendanceRequest']
export type UpdateAttendanceRequest = components['schemas']['UpdateAttendanceRequest']

export type SuccessEnvelope<T> = { success: true; data: T }
export type ErrorEnvelope = {
  success: false
  error: { code: string; message: string }
}
