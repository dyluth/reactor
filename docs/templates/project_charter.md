# **Project Charter: \[Project Name\]**

Version: 1.0  
Status: Active | Superseded | Deprecated  
Author(s): \[AI Agent Name, Your Name\]  
Date: YYYY-MM-DD

## **0\. A guide for the AI agent**

\<\!--  
AI: Your primary objective is not to complete this document, but to use it as a tool to achieve perfect clarity with the user. You are a collaborator, not an order-taker.

* **Default to questions:** If any part of the user's initial prompt is ambiguous or lacks detail, your first action must be to ask clarifying questions before attempting to fill out a section.  
* **Never invent core requirements:** Do not invent business goals, user needs, or core technical constraints. If they are missing, you must ask.  
* Validate your understanding: After filling out a significant section (like the component architecture), explicitly ask the user for confirmation and feedback.  
  \--\>

## **1\. Project overview**

*This section serves as the executive summary. It defines the project's purpose, its intended outcome, and the core problem it solves. It is the "North Star" that guides all subsequent decisions.*

### **1.1. Vision statement**

\<\!--  
AI: Provide a concise, aspirational statement that describes the long-term desired state of the world once this project is successful. What is the ultimate impact we want to achieve?  
(e.g., "To become the most trusted platform for independent artists to manage their careers.")  
\--\>

### **1.2. Mission statement**

\<\!--  
AI: Define the project's core purpose in a single, clear sentence. What are we building, who are we building it for, and why?  
(e.g., "We are building a centralised financial management tool for freelance graphic designers to simplify their invoicing and tax preparation.")  
\--\>

### **1.3. Business goals & objectives**

\<\!--  
AI: List the specific, measurable business outcomes this project is intended to achieve. These should be quantifiable and time-bound.  
If the user has not provided these, you must ask for them. For example: "What are the top 3 business goals we're trying to achieve with this project?" If the user is unsure, offer to help brainstorm some typical goals based on the project's mission statement. For example: "Based on the mission, some common goals might be user acquisition, retention, or revenue generation. Would you like to explore some of those?" Do not proceed without this information.

* **Goal 1:** (e.g., "Capture 5% of the freelance graphic designer market in the UK within two years.")  
* **Goal 2:** (e.g., "Achieve a subscription retention rate of 80% year-over-year.")  
* Goal 3: (e.g., "Become profitable within three years of launch.")  
  \--\>

## **2\. Strategic foundation**

*This section establishes the core principles and philosophies that will govern the project. It provides a framework for making consistent decisions and trade-offs.*

### **2.1. Guiding principles**

\<\!--  
AI: List the fundamental beliefs and values that will guide development. These are the project's "commandments."

* (e.g., **User-centricity:** "We will always prioritise the user's needs and experience above all else.")  
* (e.g., **Simplicity:** "We will favour simple, elegant solutions over complex ones.")  
* (e.g., **Security by design:** "Security is not an afterthought; it is a foundational requirement for every feature.")  
* (e.g., Data-informed decisions: "We will use data and user feedback, not just intuition, to guide our product roadmap.")  
  \--\>

### **2.2. Decision-making framework**

\<\!--  
AI Pitfall Alert: Without a clear hierarchy, you may optimise for the wrong thing. Use this framework to resolve trade-offs.  
AI: Define the order of priority when making difficult decisions. This clarifies what matters most when not everything can be achieved.

**Example Priority Order:**

1. **Security & Privacy:** Non-negotiable. We will never compromise user data or system security.  
2. **User Experience & Reliability:** The product must be intuitive, reliable, and solve the user's core problem effectively.  
3. **Developer Velocity:** We should be able to ship well-tested value to users quickly and sustainably.  
4. Cost: We will be mindful of infrastructure and operational costs, but not at the expense of the principles above.  
   \--\>

### **2.3. High-level success criteria**

\<\!--  
AI: Describe the key project-level metrics that will define success. These are distinct from the feature-level metrics in the Feature Design Document.

* (e.g., **Product-Market Fit:** "Achieve a Net Promoter Score (NPS) of \> 50 within the first year.")  
* (e.g., **Technical Health:** "Maintain a code test coverage of \> 90% and a P99 API latency of \< 250ms.")  
* (e.g., Operational Excellence: "Maintain a service availability of 99.95% and a mean-time-to-recovery (MTTR) of \< 15 minutes.")  
  \--\>

## **3\. Technical foundation**

*This section outlines the core technical strategy, architecture, and standards that will ensure the project is built in a consistent, scalable, and maintainable way.*

### **3.1. Core technology stack**

\<\!--  
AI: List the primary languages, frameworks, and platforms that will be used. Provide a brief justification for each major choice.  
Your recommendations must be directly justified by the project's goals, scale, and constraints. Before finalising a stack, ask the user about their team's existing expertise and operational preferences. For example: "The team has experience with Go, which makes it a strong candidate for the backend. Is that correct?"

* **Frontend:** (e.g., "React with TypeScript, chosen for its strong ecosystem and type safety.")  
* **Backend:** (e.g., "Go, chosen for its performance, simplicity, and strong concurrency model.")  
* **Database:** (e.g., "PostgreSQL, chosen for its reliability, extensibility, and rich feature set.")  
* Infrastructure: (e.g., "Docker on Google Cloud Run, chosen for its scalability and managed serverless environment.")  
  \--\>

### **3.2. High-level component architecture**

\<\!--  
AI Pitfall Alert: This is your map. Defining clear boundaries and responsibilities for each component is critical for maintainability and allows you to focus on one part of the system at a time.  
AI: Describe the major logical components or services of the system. For each component, define its core responsibility and its primary interfaces with other components. A Mermaid diagram (e.g., a component diagram) is highly encouraged here.  
After proposing an initial architecture, you must ask the user for feedback. For example: "Here is a first draft of the component architecture. Do these responsibilities and boundaries make sense to you, or is there anything that feels wrong?"  
**Example:**

* **Frontend Webapp:** A Next.js single-page application. Its sole responsibility is rendering the UI and handling user interactions. It communicates exclusively with the Public API Gateway.  
* **Public API Gateway:** A GraphQL server. It is the single entry point for all client applications. It authenticates requests and routes them to the appropriate internal services.  
* **Users Service:** A Go service. Manages all user data, authentication, and authorisation logic. Exposes a gRPC API for internal use.  
* Invoicing Service: A Go service. Manages all logic related to creating, sending, and tracking invoices. Exposes a gRPC API.  
  \--\>

### **3.3. Architectural principles**

\<\!--  
AI: Define the high-level architectural approach and patterns that will be followed.

* (e.g., **12-Factor App:** "We will adhere to the 12-Factor App methodology to build a scalable and resilient cloud-native application.")  
* (e.g., **API-first design:** "The backend will expose a clean, well-documented API that the frontend (and future clients) will consume.")  
* (e.g., **Stateless services:** "Backend services should be stateless wherever possible to simplify scaling and improve resilience.")  
* (e.g., Infrastructure as Code (IaC): "All infrastructure will be defined and managed in code using Terraform.")  
  \--\>

### **3.4. Coding standards & conventions**

\<\!--  
AI: Specify the tools and standards that will be used to maintain code quality and consistency.

* **Linting:** (e.g., "ESLint with the Airbnb config for the frontend; golangci-lint for the backend.")  
* **Formatting:** (e.g., "Prettier for the frontend; gofmt for the backend. Formatting will be enforced by a pre-commit hook.")  
* Naming Conventions: (e.g., "API endpoints will use kebab-case; database tables will use snake\_case.")  
  \--\>

## **4\. Execution framework**

*This section defines the processes and quality gates that govern how work is delivered.*

### **4.1. Definition of done (DoD)**

\<\!--  
AI: This is a critical checklist. No feature or piece of work is considered complete until it meets every one of these criteria. This is the project's universal quality standard.  
Before treating this list as final, review it with the user. Ask: "This is a standard Definition of Done. Should we add, remove, or change anything on this list to better fit the needs of this project?"  
A feature is **done** only when it:

* \[ \] Meets all acceptance criteria defined in its user stories.  
* \[ \] Has comprehensive unit and integration tests with \> 90% code coverage.  
* \[ \] Has a corresponding end-to-end test for each user story.  
* \[ \] Passes all CI/CD pipeline checks, including linting, formatting, and security scans.  
* \[ \] Is fully compliant with the project's accessibility (a11y) standards.  
* \[ \] Is deployed behind a feature flag (if applicable).  
* \[ \] Has been documented for end-users (if applicable) and for developers in the docs/ directory.  
* \[ \] Has the necessary monitoring, logging, and alerting in place.  
* \[ \] Has been peer-reviewed and approved by at least one other developer (or the project lead).  
  \--\>

### **4.2. Development workflow**

\<\!--  
AI: Describe the end-to-end process from idea to production.  
(e.g., "Development will follow a trunk-based development model. All work will be done on short-lived feature branches that are merged directly into main after a successful PR review. The main branch is continuously deployed to a staging environment, and manual deployments to production are triggered from main via a Git tag.")  
\--\>

### **4.3. Team & communication**

\<\!--  
AI: Outline the primary methods of communication.

* **Primary Channel:** (e.g., "Asynchronous communication via GitHub Issues and Pull Requests is preferred.")  
* **Stand-ups:** (e.g., "A daily summary of progress and blockers will be posted in the \#project-updates Slack channel.")  
* Source of Truth: "This document and the code itself are the ultimate sources of truth."  
  \--\>

## **5\. Decision log**

*This section is a living appendix used to record significant architectural decisions, changes in strategy, or important learnings over the project's lifetime.*

\<\!--  
AI: As the project evolves, record major decisions here to provide context for future development.  
**YYYY-MM-DD: Decision to switch from REST to GraphQL for the public API**

* **Context:** (e.g., "Our initial REST API was becoming difficult to manage as the number of client-side data requirements grew.")  
* **Justification:** (e.g., "GraphQL allows clients to request exactly the data they need, reducing over-fetching and improving performance. This aligns with our 'User Experience' priority.")  
* Impact: (e.g., "All new public-facing data endpoints will be developed using GraphQL. Existing REST endpoints will be deprecated over the next 6 months.")  
  \--\>