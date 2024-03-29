describe('container-e2e', () => {
  beforeEach(() => cy.visit('/'));

  it('should display', () => {
    cy.contains('JDW');
  });
});
