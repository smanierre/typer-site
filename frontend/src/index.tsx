import * as React from "react";
import ReactDOM from "react-dom";
import { BrowserRouter as Router, Switch, Route } from "react-router-dom";

import Landing from "./Landing/Landing.tsx";
import InterfaceResults from "./InterfaceResults/InterfaceResults";
import TypeResults from "./TypeResults/TypeResults";
import GlobalStyles from "./GlobalStyles";

const Index: React.FC = () => {
  return (
    <>
      <GlobalStyles />
      <Router>
        <Switch>
          <Route exact path="/">
            <Landing />
          </Route>
          <Route
            path="/results/interface/:query"
            render={routeProps => {
              let query: string = "";
              if (typeof routeProps.match.params.query === "string") {
                query = routeProps.match.params.query;
              }
              return <InterfaceResults searchQuery={query} />;
            }}
          />
          <Route
            path="/results/type/:query"
            render={routeProps => {
              let query: string = "";
              if (typeof routeProps.match.params.query === "string") {
                query = routeProps.match.params.query;
              }
              return <TypeResults searchQuery={query} />;
            }}
          />
        </Switch>
      </Router>
    </>
  );
};

ReactDOM.render(<Index />, document.getElementById("root"));
