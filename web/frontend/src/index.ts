// Theme (inject CSS custom properties)
import { haTheme } from "./theme";

// Inject theme into document
const style = document.createElement("style");
style.textContent = haTheme.cssText;
document.head.appendChild(style);

// Components
import "./components/ms-card";
import "./components/ms-dialog";
import "./components/ms-button";
import "./components/ms-alert";
import "./components/ms-status-dot";
import "./components/ms-form";
import "./components/ms-sidebar";
import "./components/ms-toast";
import "./components/ms-app-shell";

// Views
import "./views/ms-view-dashboard";
import "./views/ms-view-devices";
import "./views/ms-view-gateways";
import "./views/ms-view-mqtt";
import "./views/ms-view-mqtt-topics";
import "./views/ms-view-aliases";
import "./views/ms-view-editor";
import "./views/ms-view-logs";
import "./views/ms-view-traffic";

// WebSocket
import { adapterWS } from "./websocket";

// Connect WebSocket
adapterWS.connect();
