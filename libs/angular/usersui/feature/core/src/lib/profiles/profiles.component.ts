import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ProfilesService } from '@jdw/angular-usersui-data-access';
import {
  dateFilterComparator,
  dateSortComparator,
} from '@jdw/angular-usersui-util';
import { ColDef, GridOptions } from 'ag-grid-community';
import { AgGridAngular } from 'ag-grid-angular';
import { ProfilesActionButtonCellRendererComponent } from '../profiles-action-button-cell-renderer/profiles-action-button-cell-renderer.component';
import { Profile } from '@jdw/angular-shared-util';

@Component({
  selector: 'lib-profiles',
  imports: [CommonModule, AgGridAngular],
  templateUrl: './profiles.component.html',
  styleUrl: './profiles.component.scss',
})
export class ProfilesComponent implements OnInit {
  private profilesService: ProfilesService = inject(ProfilesService);
  loading = true;
  profiles: Profile[] = [];

  colDefs: ColDef[] = [
    { field: 'id', headerName: 'ID', cellDataType: 'number' },
    { field: 'firstName', headerName: 'First Name', cellDataType: 'text' },
    { field: 'middleName', headerName: 'Middle Name', cellDataType: 'text' },
    { field: 'lastName', headerName: 'Last Name', cellDataType: 'text' },
    {
      field: 'birthdate',
      headerName: 'Birthdate',
      cellDataType: 'text',
      filter: 'agDateColumnFilter',
      filterParams: {
        comparator: dateFilterComparator,
      },
    },
    { field: 'userId', headerName: 'User ID', cellDataType: 'text' },
    {
      field: 'addresses',
      headerName: 'Address Count',
      valueGetter: (params) => params.data.addresses?.length || 0,
      cellDataType: 'number',
    },
    {
      field: 'icon',
      headerName: 'Has Icon',
      valueGetter: (params) => (params.data.icon ? 'Yes' : 'No'),
      cellDataType: 'text',
    },
    {
      field: 'createdByUserId',
      headerName: 'Created By',
      cellDataType: 'number',
    },
    {
      field: 'createdTime',
      headerName: 'Created Time',
      filter: 'agDateColumnFilter',
      filterParams: {
        comparator: dateFilterComparator,
      },
      comparator: dateSortComparator,
      cellDataType: 'text',
    },
    {
      field: 'modifiedByUserId',
      headerName: 'Modified By',
      cellDataType: 'number',
    },
    {
      field: 'modifiedTime',
      headerName: 'Modified Time',
      filter: 'agDateColumnFilter',
      filterParams: {
        comparator: dateFilterComparator,
      },
      comparator: dateSortComparator,
      cellDataType: 'text',
    },
    {
      field: 'actions',
      headerName: 'Actions',
      cellRenderer: ProfilesActionButtonCellRendererComponent,
      maxWidth: 172,
      minWidth: 172,
      resizable: false,
      filter: false,
      sortable: false,
    },
  ];

  defaultColDef: ColDef = {
    sortable: true,
    filter: true,
    resizable: true,
    cellStyle: {
      display: 'flex',
      alignItems: 'center',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
      overflow: 'hidden',
    },
  };

  gridOptions: GridOptions = {
    loading: this.loading,
    columnDefs: this.colDefs,
    defaultColDef: this.defaultColDef,
    pagination: true,
    rowSelection: {
      mode: 'multiRow',
      checkboxes: false,
      headerCheckbox: false,
      enableClickSelection: true,
    },
    animateRows: true,
    enableCellTextSelection: true,
    autoSizeStrategy: {
      type: 'fitCellContents',
    },
    suppressColumnVirtualisation: true,
  };

  ngOnInit(): void {
    this.profilesService.getProfiles().subscribe({
      next: (response) => {
        this.profiles = response;
        this.loading = false;
      },
    });
  }
}
