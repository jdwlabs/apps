describe('usersui-e2e', () => {
  beforeEach(() => cy.visit('/'));

  it('should be running', () => {
    cy.url().should('contain', '/');
  });
});
