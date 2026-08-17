# Architecture

## Overview

flgr is designed as a traditional web application with a front-end (client) and a back-end (server) that communicate over HTTP. The back-end is responsible for managing feature flags, user profiles, and service keys, while the front-end provides a user interface for administrators to manage these resources.

## Technology Stack

- **Front-end:** React.js for building the user interface, with Redux for state management.
- **Back-end:** Golang for building the RESTful API, with GIN for routing and handling HTTP requests.
- **Database:** SQLite for storing feature flags, user profiles, and service keys.