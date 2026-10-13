import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./index.css";
import { subscribeToEngagementEvents } from "./lib/query";

subscribeToEngagementEvents();

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
