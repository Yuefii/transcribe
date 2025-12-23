
## Architecture System

![Architecture System](.github/assets/architecture.PNG)

## Feature System

- JWT Authentication
- File upload
- Redis queue system
- Async processing with Python worker
- Status tracking (queued → processing → done/failed)

## API Endpoints

- POST `/api/auth/sign-up`
- POST `/api/auth/sign-in`
- GET `/api/user/profile`
- POST `/api/transcribe`
- GET `/api/transcribe/:job_id`
- GET `/api/transcribe`
- DELETE `/api/transcribe/:job_id`