/// <reference types="cypress" />

// ***********************************************
// This example commands.ts shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************

// eslint-disable-next-line @typescript-eslint/no-namespace
declare namespace Cypress {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  interface Chainable<Subject> {
    getByCy(selector: string, ...args: never[]): Cypress.Chainable<never>;
  }
}

Cypress.Commands.add('getByCy', (selector, ...args) => {
  return cy.get(`[data-cy=${selector}]`, ...args);
});
